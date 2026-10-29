package defaultbootstrap

import (
	"bytes"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"io"
	"net"
	"strings"

	"github.com/sekret01/sekret_go_proxy/internal/bootstrap"
	"github.com/sekret01/sekret_go_proxy/internal/core"
	"golang.org/x/crypto/hkdf"
)

type HelloMsg struct {
	Version uint16
	Random  [32]byte
	PubKey  [32]byte
	Steps   []string
}

// можно изменять / вынести отдельно для изменения (core.types)
var bootstrapMagicBytes []byte = []byte{0x53, 0x24, 0x68, 0x12}

const (
	magicBytesLength  = 4
	payloadSizeLength = 4
	versionSize       = 2
	randomSize        = 32
	pubKeySize        = 32
)

func parseHelloMsgHeader(data []byte) (int, error) {
	if len(data) < magicBytesLength+payloadSizeLength {
		return -1, core.ErrInsufficientHeaderLensth
	}
	step := 0
	magicBytes := data[step : step+magicBytesLength]
	if !compareBootstrapMagic(magicBytes) {
		return -1, core.ErrInvalidMagic
	}
	step += magicBytesLength
	payloadSize := binary.BigEndian.Uint32(data[step : step+payloadSizeLength])
	return int(payloadSize), nil
}

func parseHelloMsgPayload(data []byte) (HelloMsg, error) {
	step := 0
	version := binary.BigEndian.Uint16(data[step : step+versionSize])
	step += versionSize
	random := data[step : step+randomSize]
	step += randomSize
	pubkey := data[step : step+pubKeySize]
	step += pubKeySize
	stepStringBytes := data[step:]
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
	payloadSize := uint32(versionSize + len(msg.Random) + len(msg.PubKey) + len(stepStringBytes))

	buf.Write(bootstrapMagicBytes)
	binary.Write(buf, binary.BigEndian, payloadSize)
	binary.Write(buf, binary.BigEndian, msg.Version)
	buf.Write(msg.Random[:])
	buf.Write(msg.PubKey[:])
	buf.Write(stepStringBytes)

	return buf.Bytes(), nil
}

func compareBootstrapMagic(magic []byte) bool {
	if len(magic) != len(bootstrapMagicBytes) {
		return false
	}
	for i := range len(bootstrapMagicBytes) {
		if magic[i] != bootstrapMagicBytes[i] {
			return false
		}
	}
	return true
}

type BaseBootstrap struct {
	cfg *bootstrap.BootstrapConfig
}

func (b *BaseBootstrap) ClientBootstrap(con net.Conn) (steps []string, sessionKey []byte, err error) {
	curve := ecdh.X25519()
	clientPrivateKey, _ := curve.GenerateKey(rand.Reader)
	clientPublicKey := clientPrivateKey.PublicKey()
	clientRandom := make([]byte, 32)
	_, _ = io.ReadFull(rand.Reader, clientRandom)

	recMsg := HelloMsg{
		Version: 1,
		Random:  [32]byte(clientRandom),
		PubKey:  [32]byte(clientPublicKey.Bytes()),
		Steps:   []string{},
	}
	bytesData, err := helloMsgToBytes(recMsg)
	if err != nil {
		return nil, nil, err
	}
	con.Write(bytesData)

	// keys
	msg, err := b.readBootstrapMsg(con)
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
	msg, err := b.readBootstrapMsg(con)
	if err != nil {
		return nil, nil, err
	}
	curve := ecdh.X25519()
	serverPrivateKey, _ := curve.GenerateKey(rand.Reader)
	serverPublicKey := serverPrivateKey.PublicKey()
	serverRandom := make([]byte, 32)
	_, _ = io.ReadFull(rand.Reader, serverRandom)
	recMsg := HelloMsg{
		Version: 1,
		Random:  [32]byte(serverRandom),
		PubKey:  [32]byte(serverPublicKey.Bytes()),
		Steps:   b.cfg.HandshakeSteps,
	}
	bytesData, err := helloMsgToBytes(recMsg)
	if err != nil {
		return nil, nil, err
	}
	con.Write(bytesData)

	// keys
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

func (b *BaseBootstrap) readBootstrapMsg(con net.Conn) (*HelloMsg, error) {
	headerBuf := make([]byte, magicBytesLength+payloadSizeLength)
	if _, err := io.ReadFull(con, headerBuf); err != nil {
		return nil, err
	}
	payloadSize, err := parseHelloMsgHeader(headerBuf)
	if err != nil {
		return nil, err
	}
	if payloadSizeLength < 0 || payloadSize > core.MAX_PAYLOAD_SIZE {
		return nil, core.ErrPayloadSizeTooLarge
	}

	payloadBuf := make([]byte, payloadSize)
	if _, err = io.ReadFull(con, payloadBuf); err != nil {
		return nil, err
	}
	msg, err := parseHelloMsgPayload(payloadBuf)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

func NewBaseBootstrap(cfg *bootstrap.BootstrapConfig) (core.Bootstrap, error) {
	return &BaseBootstrap{
		cfg: cfg,
	}, nil
}

func init() {
	bootstrap.Registrate("default", NewBaseBootstrap)
}
