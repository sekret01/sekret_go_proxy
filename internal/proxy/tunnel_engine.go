package proxy

import (
	"errors"
	"net"
	"sync"
	"time"

	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/utils"
	"github.com/sekret01/sekret_go_proxy/pkg/logger"
)

type ClientListener interface {
	CloseListen()
}

// Функции взаимодействия с тунелем
type TunnelEngine struct {
	transport core.Transport // Подключение и передача информации
	encryptor core.Encryptor // Шифрование и расшифровка
	framer    core.Framer    // Оборачивание и парсинг самописных протоколов
	auth      core.Auth

	remoteAddr string // Адрес удаленного узла для подключения

	logger          logger.Logger
	mutex           sync.Mutex
	tunnelChan      chan []byte
	clientListeners []ClientListener
	tunnelMessages  chan core.MessageData

	serverTonnelConn net.Conn      // Туннельное подключение к серверу
	status           *TunnelStatus // Состояние работы
}

func (t *TunnelEngine) ConnectToTunnel() error {
	t.status.SetLaunch()
	conn, err := t.waitConnectionToTunnel()
	if err != nil {
		t.status.SetStopped()
		t.logger.Warning("[ClientTunnel] :: connect remote addr -> " + err.Error())
		return err
	}
	t.logger.Info("Try authenticate")
	_, err = t.auth.ClientHandshake(conn)
	if err != nil {
		t.status.SetStopped()
		t.logger.Error(err.Error())
		return err
	}
	t.status.SetRunning()
	t.serverTonnelConn = conn
	go t.tunnelChanReader()
	go t.tunnelReader()
	return nil
}

func (t *TunnelEngine) RegistrateListener(listener ClientListener) {
	t.clientListeners = append(t.clientListeners, listener)
}

func (t *TunnelEngine) IsRunning() bool {
	return t.status.isRunning
}

func (t *TunnelEngine) GetStatus() *TunnelStatus {
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
	if !t.status.isRunning {
		t.logger.Warning("[stopListening] Trying to stop stopped service, cencel")
		return nil
	}
	if stopRunning {
		t.status.SetStopped()
	}
	if t.serverTonnelConn != nil {
		t.serverTonnelConn.Close()
		t.serverTonnelConn = nil
	}
	t.closeListeners()
	close(t.tunnelChan)
	t.tunnelChan = make(chan []byte, 1000)
	close(t.tunnelMessages)
	t.tunnelMessages = make(chan core.MessageData, 1000)
	t.logger.Info("[stopListening] Stop tunnel listening")
	return nil
}

// Чтение, обработка и перессылка данных с сервера
func (t *TunnelEngine) tunnelReader() {
	t.logger.Info("[tunnelReader] Start tunnel listening")

	for t.status.isRunning {
		frame, err := ReadFrameFromConnection(t.serverTonnelConn, t.framer)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				t.logger.Warning("[tunnelReader] Close tunel reader: (" + err.Error() + ")")
			} else {
				t.logger.Error("[tunnelReader] ERROR in read frame: " + err.Error())
			}
			t.stopListening(false)
			return
		}
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
	t.logger.Error("EXIT FROM READ CYCLE")
}

func (t *TunnelEngine) sendIntoTunnel(requestId core.RequestID, msgType core.MessageType, payload []byte) error {
	if !t.status.isRunning || t.serverTonnelConn == nil {
		return nil
	}
	t.logger.Debug("[sendIntoTunnel] Prepeare new msg: requestId: [" + utils.RequestIdToString(requestId) + "], msgType: [" + utils.MessageTypeToHexString(msgType) + "], msg: [" + utils.BytesToString(payload, 20) + "]")
	payloadEncode := t.encryptor.Encrypt([]byte(payload))
	frame, err := t.framer.Frame(payloadEncode, msgType, requestId)
	if err != nil {
		t.logger.Error("[sendIntoTunnel] Error in send data: " + err.Error())
		return err
	}
	t.mutex.Lock()
	t.tunnelChan <- frame
	t.mutex.Unlock()
	return nil
}

func (t *TunnelEngine) sendCloseIntoTunnel(requestId core.RequestID) {
	t.logger.Debug("[sendCloseIntoTunnel] close " + utils.RequestIdToString(requestId))
	t.sendIntoTunnel(requestId, core.MsgClose, []byte{})
}

func (t *TunnelEngine) tunnelChanReader() {
	for frame := range t.tunnelChan {
		if t.serverTonnelConn == nil {
			continue
		}
		_, err := t.serverTonnelConn.Write(frame)
		if err != nil {
			t.logger.Error("[tunnelChanReader] Error in send data: " + err.Error())
			continue
		}
		t.logger.Debug("[tunnelChanReader] Data has been sent")
	}
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
	auth core.Auth) *TunnelEngine {
	engine := &TunnelEngine{
		transport:        transport,
		encryptor:        encryptor,
		framer:           framer,
		status:           NewTunnelStatus(),
		serverTonnelConn: nil,
		remoteAddr:       cfg.RemoteHost,
		logger:           logger,
		tunnelChan:       make(chan []byte, 1000),
		clientListeners:  make([]ClientListener, 0),
		tunnelMessages:   make(chan core.MessageData, 1000),
		auth:             auth,
	}
	return engine
}
