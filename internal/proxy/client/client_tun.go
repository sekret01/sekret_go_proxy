package client

import (
	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/proxy"
	"github.com/sekret01/sekret_go_proxy/internal/utils"
	"github.com/sekret01/sekret_go_proxy/pkg/logger"
)

type ClientTun struct {
	tunnelEngine *TunnelEngine
	packetDevice core.PacketDevice
	logger       logger.Logger
}

func (c *ClientTun) Start() error {
	if c.tunnelEngine.IsRunning() {
		c.logger.Warning("[Start] Trying to start running service, return")
		return nil
	}
	c.tunnelEngine.SetLaunch()

	err := c.packetDevice.CreateTun("tun0")
	if err != nil {
		c.logger.Error("[Start] error: " + err.Error())
		c.tunnelEngine.SetStopped()
		return err
	}
	go c.tunnelMessagesReader()
	for c.tunnelEngine.IsRunning() {
		err := c.tunnelEngine.ConnectToTunnel()
		if err != nil {
			c.Stop()
			return err
		}
		c.tunReader()

	}
	return nil
}

func (c *ClientTun) Stop() error {
	return c.tunnelEngine.stopListening(true)
}

func (c *ClientTun) GetStatus() *proxy.TunnelStatus {
	return c.tunnelEngine.GetStatus()
}

func (c *ClientTun) CloseListen() {
	c.packetDevice.Close()
}

func (c *ClientTun) tunReader() {
	buf := make([]byte, c.packetDevice.MTU())
	for c.tunnelEngine.IsRunning() {
		n, err := c.packetDevice.ReadPacket(buf)
		if err != nil {
			c.logger.Error("Error in read tun: " + err.Error())
			return
		}
		data := buf[:n]
		c.logger.Debug("[tunReader] get [" + utils.IntToString(n) + "] bytes: " + utils.BytesToString(data, 20))
		c.tunnelEngine.sendIntoTunnel(core.GetNoopID(), core.MsgPacket, data)
		// time.Sleep(time.Second)
	}
}

func (c *ClientTun) tunnelMessagesReader() {
	for msg := range c.tunnelEngine.tunnelMessages {
		if msg.MsgType != core.MsgPacket {
			continue
		}
		_, err := c.packetDevice.WritePacket(msg.Data)
		if err != nil {
			c.logger.Warning("[tunnelMessagesReader] write data error: " + err.Error())
			continue
		}
		c.logger.Debug("[tunnelMessagesReader] write data for [ " + utils.RequestIdToString(msg.RequestId) + " ]: " + utils.BytesToString(msg.Data, 20))
	}
}

func NewClientTun(
	tunnelEngin *TunnelEngine,
	packetsDevice core.PacketDevice,
	logger logger.Logger,
) *ClientTun {
	client := &ClientTun{
		tunnelEngine: tunnelEngin,
		packetDevice: packetsDevice,
		logger:       logger,
	}
	tunnelEngin.RegistrateListener(client)
	return client
}
