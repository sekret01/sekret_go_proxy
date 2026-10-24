package bootstrap

import (
	"bytes"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"io"
	"net"
	"strings"

	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
	"golang.org/x/crypto/hkdf"
)

type HelloMsg struct {
	Version uint16
	Random  [32]byte
	PubKey  [32]byte
	Steps   []string
}

func getMsgSize() int {
	return 100
}

func parseHelloMsg(data []byte) (HelloMsg, error) {
	step := 0
	payloadSize := binary.BigEndian.Uint32(data[0:4])
	step += 4
	version := binary.BigEndian.Uint16(data[step : step+2])
	step += 2
	random := data[step : step+32]
	step += 32
	pubkey := data[step : step+32]
	step += 32
	stepStringBytes := data[step : payloadSize+4]
	steps := strings.Split(string(stepStringBytes), ",")
	msg := HelloMsg{
		Version: version,
		Random:  [32]byte(random),
		PubKey:  [32]byte(pubkey),
		Steps:   steps,
	}
	return msg, nil
}

func helloMsgToBytes(msg HelloMsg) ([]byte, error) {
	buf := new(bytes.Buffer)
	stepStringBytes := []byte(strings.Join(msg.Steps, ","))
	payloadSize := uint32(2 + len(msg.Random) + len(msg.PubKey) + len(stepStringBytes))

	binary.Write(buf, binary.BigEndian, payloadSize)
	binary.Write(buf, binary.BigEndian, msg.Version)
	buf.Write(msg.Random[:])
	buf.Write(msg.PubKey[:])
	buf.Write(stepStringBytes)

	return buf.Bytes(), nil
}

type BaseBootstrap struct {
	cfg *config.Config
}

func (b *BaseBootstrap) ClientBootstrap(con net.Conn) (steps []string, sessionKey []byte, err error) {
	curve := ecdh.X25519()
	clientPrivateKey, _ := curve.GenerateKey(rand.Reader)
	clientPublicKey := clientPrivateKey.PublicKey()
	clientRandom := make([]byte, 32)
	_, _ = io.ReadFull(rand.Reader, clientRandom)

	msg := HelloMsg{
		Version: 1,
		Random:  [32]byte(clientRandom),
		PubKey:  [32]byte(clientPublicKey.Bytes()),
		Steps:   []string{},
	}
	bytesData, err := helloMsgToBytes(msg)
	if err != nil {
		return nil, nil, err
	}
	con.Write(bytesData)

	// keys
	buf := make([]byte, 100)
	n, err := con.Read(buf)
	if err != nil {
		return nil, nil, err
	}
	msg, err = parseHelloMsg(buf[:n])
	if err != nil {
		return nil, nil, err
	}
	serverPubKey, err := curve.NewPublicKey(msg.PubKey[:])
	if err != nil {
		return nil, nil, err
	}
	clientSharedSecret, err := clientPrivateKey.ECDH(serverPubKey)
	if err != nil {
		return nil, nil, err
	}
	salt := append(clientRandom, msg.Random[:]...)
	info := []byte("info for reader")
	hkdfReader := hkdf.New(sha256.New, clientSharedSecret, salt, info)
	clientWriteKey := make([]byte, 32)
	_, err = io.ReadFull(hkdfReader, clientWriteKey)
	if err != nil {
		return nil, nil, err
	}
	return msg.Steps, clientWriteKey, nil

}
func (b *BaseBootstrap) ServerBootstrap(con net.Conn) (steps []string, sessionKey []byte, err error) {
	buf := make([]byte, 100)
	n, err := con.Read(buf)
	if err != nil {
		return nil, nil, err
	}
	curve := ecdh.X25519()
	serverPrivateKey, _ := curve.GenerateKey(rand.Reader)
	serverPublicKey := serverPrivateKey.PublicKey()
	serverRandom := make([]byte, 32)
	_, _ = io.ReadFull(rand.Reader, serverRandom)
	msg := HelloMsg{
		Version: 1,
		Random:  [32]byte(serverRandom),
		PubKey:  [32]byte(serverPublicKey.Bytes()),
		Steps:   b.cfg.HandshakeSteps,
	}
	bytesData, err := helloMsgToBytes(msg)
	if err != nil {
		return nil, nil, err
	}
	con.Write(bytesData)

	// keys
	msg, err = parseHelloMsg(buf[:n])
	if err != nil {
		return nil, nil, err
	}
	clientPubKey, err := curve.NewPublicKey(msg.PubKey[:])
	if err != nil {
		return nil, nil, err
	}
	serverSharedSecret, err := serverPrivateKey.ECDH(clientPubKey)
	if err != nil {
		return nil, nil, err
	}
	salt := append(msg.Random[:], serverRandom...)
	info := []byte("info for reader")
	hkdfReader := hkdf.New(sha256.New, serverSharedSecret, salt, info)
	serverWriteKey := make([]byte, 32)
	_, err = io.ReadFull(hkdfReader, serverWriteKey)
	if err != nil {
		return nil, nil, err
	}
	return b.cfg.HandshakeSteps, serverWriteKey, nil

}

func NewBaseBootstrap(cfg *config.Config) core.Bootstrap {
	return &BaseBootstrap{
		cfg: cfg,
	}
}
