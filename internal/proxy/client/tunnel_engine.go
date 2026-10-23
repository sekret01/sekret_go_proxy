package client

import (
	"net"
	"sync"
	"time"

	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/encryptors"
	"github.com/sekret01/sekret_go_proxy/internal/proxy"
	"github.com/sekret01/sekret_go_proxy/internal/utils"
	"github.com/sekret01/sekret_go_proxy/pkg/logger"
)

type ClientListener interface {
	CloseListen()
}

// Функции взаимодействия с тунелем
type TunnelEngine struct {
	tunnelNode *proxy.TunnelNode
	transport  core.Transport // Подключение и передача информации
	encryptor  core.Encryptor // Шифрование и расшифровка
	framer     core.Framer    // Оборачивание и парсинг самописных протоколов
	auth       core.Auth
	bootstrap  core.Bootstrap

	remoteAddr string // Адрес удаленного узла для подключения

	logger          logger.Logger
	mutex           sync.Mutex
	clientListeners []ClientListener
	tunnelMessages  chan core.MessageData

	status *proxy.TunnelStatus // Состояние работы
	ctx    *core.HandshakeContext
	cfg    *config.Config
}

func (t *TunnelEngine) ConnectToTunnel() error {
	t.status.SetLaunch()
	conn, err := t.waitConnectionToTunnel()
	if err != nil {
		t.status.SetStopped()
		t.logger.Warning("[ConnectToTunnel] :: connect remote addr -> " + err.Error())
		return err
	}
	t.logger.Info("Try authenticate")
	if err := t.authenticate(conn); err != nil {
		t.status.SetStopped()
		return nil
	}
	t.logger.Debug("[ConnectToTunnel] auth success")
	// _, key, err := t.bootstrap.ClientBootstrap(conn)
	// enc, err := encryptors.NewEncryptor(t.cfg.EncryptorType, key, t.cfg)
	// if err != nil {
	// 	t.logger.Warning("[listenConnectoins] Bootstrap encoder create error: " + err.Error())
	// 	return err
	// }
	// t.auth.SetEncryptor(enc)
	// t.ctx, err = t.auth.ClientHandshake(conn)
	// if err != nil {
	// 	t.status.SetStopped()
	// 	t.logger.Error(err.Error())
	// 	return err
	// }
	t.status.SetRunning()
	t.tunnelNode = proxy.NewTunnelNode(conn, t.framer)
	go t.tunnelReader()
	return nil
}

func (t *TunnelEngine) GetHandshakeContext() *core.HandshakeContext {
	return t.ctx
}

func (t *TunnelEngine) RegistrateListener(listener ClientListener) {
	t.clientListeners = append(t.clientListeners, listener)
}

func (t *TunnelEngine) IsRunning() bool {
	return t.status.IsRunning()
}

func (t *TunnelEngine) GetStatus() *proxy.TunnelStatus {
	return t.status
}

func (t *TunnelEngine) SetRunning() {
	t.status.SetRunning()
}

func (t *TunnelEngine) SetLaunch() {
	t.status.SetLaunch()
}

func (t *TunnelEngine) SetStopped() {
	t.status.SetStopped()
}

func (t *TunnelEngine) authenticate(conn net.Conn) error {
	_, key, err := t.bootstrap.ClientBootstrap(conn)
	enc, err := encryptors.NewEncryptor(t.cfg.EncryptorType, key, t.cfg)
	if err != nil {
		t.logger.Warning("[authenticate] Bootstrap encoder create error: " + err.Error())
		return err
	}
	t.auth.SetEncryptor(enc)
	t.ctx, err = t.auth.ClientHandshake(conn)
	if err != nil {
		t.logger.Error("[authenticate] ClientHandshake error: " + err.Error())
		return err
	}
	return nil
}

func (t *TunnelEngine) waitConnectionToTunnel() (net.Conn, error) {
	t.logger.Info("[WatiConnection] :: waiting tunnel connection")
	for t.status.IsRunning() {
		conn, err := t.transport.Dial(t.remoteAddr)
		if err == nil {
			t.logger.Info("[WatiConnection] :: tunnel found")
			return conn, nil
		}
		time.Sleep(time.Second * 5) // TODO вынести в конфиг
	}
	return nil, core.ErrAppNotRunning
}

func (t *TunnelEngine) stopListening(stopRunning bool) error {
	if !t.status.IsRunning() {
		t.logger.Warning("[stopListening] Trying to stop stopped service, cencel")
		return nil
	}
	if stopRunning {
		t.status.SetStopped()
	}
	t.sendCloseIntoTunnel(core.GetNoopID()) // TODO генерировать 1 раз вначале, хранить в структуре
	t.tunnelNode.Stop()
	t.closeListeners()
	close(t.tunnelMessages)                              // TODO нужно ли
	t.tunnelMessages = make(chan core.MessageData, 1000) // TODO нужно ли
	t.logger.Info("[stopListening] Stop tunnel listening")
	return nil
}

// Чтение, обработка и перессылка данных с сервера
func (t *TunnelEngine) tunnelReader() {
	t.logger.Info("[tunnelReader] Start tunnel listening")
	readChannel := t.tunnelNode.GetReadChan().GetChannel()

	for {
		select {
		case <-t.tunnelNode.Done():
			t.logger.Info("[tunnelReader] Exit from read cycle")
			if t.status.IsRunning() {
				t.stopListening(false)
			}
			return
		case frame := <-readChannel:
			data, msgType, requestId, err := t.framer.Unframe(frame)
			if err != nil {
				t.logger.Error("[tunnelReader] ERROR: unframe: " + err.Error())
				continue
			}
			decryptData := t.encryptor.Decrypt(data)
			frameData := core.MessageData{
				MsgType:   msgType,
				RequestId: requestId,
				Data:      decryptData,
			}
			t.tunnelMessages <- frameData
		}

	}

}

func (t *TunnelEngine) sendIntoTunnel(requestId core.RequestID, msgType core.MessageType, payload []byte) error {
	if !t.status.IsRunning() {
		return nil
	}
	t.logger.Debug("[sendIntoTunnel] Prepeare new msg: requestId: [" + utils.RequestIdToString(requestId) + "], msgType: [" + utils.MessageTypeToHexString(msgType) + "], msg: [" + utils.BytesToString(payload, 100) + "]")
	payloadEncode := t.encryptor.Encrypt([]byte(payload))
	frame, err := t.framer.Frame(payloadEncode, msgType, requestId)
	if err != nil {
		t.logger.Error("[sendIntoTunnel] Error in send data: " + err.Error())
		return err
	}
	t.tunnelNode.Send(frame)
	return nil
}

func (t *TunnelEngine) sendCloseIntoTunnel(requestId core.RequestID) {
	t.logger.Debug("[sendCloseIntoTunnel] close " + utils.RequestIdToString(requestId))
	t.sendIntoTunnel(requestId, core.MsgClose, []byte{})
}

func (t *TunnelEngine) closeListeners() {
	for _, listener := range t.clientListeners {
		listener.CloseListen()
	}
}

func NewTunnelEngine(
	transport core.Transport,
	encryptor core.Encryptor,
	framer core.Framer,
	cfg *config.Config,
	logger logger.Logger,
	auth core.Auth,
	bootstrap core.Bootstrap,
) *TunnelEngine {
	engine := &TunnelEngine{
		tunnelNode:      nil,
		transport:       transport,
		encryptor:       encryptor,
		framer:          framer,
		status:          proxy.NewTunnelStatus(),
		remoteAddr:      cfg.RemoteHost,
		logger:          logger,
		clientListeners: make([]ClientListener, 0),
		tunnelMessages:  make(chan core.MessageData, 1000),
		auth:            auth,
		bootstrap:       bootstrap,
		cfg:             cfg,
	}
	return engine
}
