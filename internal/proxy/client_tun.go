package proxy

import (
	"fmt"
	"time"

	"github.com/sekret01/sekret_go_proxy/internal/core"
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
	for c.tunnelEngine.IsRunning() {
		err := c.tunnelEngine.ConnectToTunnel()
		if err != nil {
			c.CloseListen()
			return err
		}
		c.tunReader()

	}
	return nil
}

func (c *ClientTun) Stop() error {
	c.tunnelEngine.stopListening(true)
	return c.CloseListen()
}

func (c *ClientTun) GetStatus() *TunnelStatus {
	return c.tunnelEngine.GetStatus()
}

func (c *ClientTun) CloseListen() error {
	return c.packetDevice.Close()
}

func (c *ClientTun) tunReader() {
	for {
		buf := make([]byte, c.packetDevice.MTU())
		data, err := c.packetDevice.ReadPacket(buf)
		if err != nil {
			c.logger.Error("Error in read tun: " + err.Error())
			return
		}
		fmt.Printf("[>] (%d) %s\n", data, buf)
		time.Sleep(time.Second)
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
	return client
}
