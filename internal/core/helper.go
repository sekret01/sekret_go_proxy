package core

type HandshakeContextGenerator interface {
	NewHandshakeContext() *HandshakeContext
}

type HandshakeGeneratorSubscriber interface {
	SubscribeGenerator(*HandshakeContextGenerator)
}

type HandshakeGeneratorConnector struct {
	generator HandshakeContextGenerator
}

func (c *HandshakeGeneratorConnector) SubscribeGenerator(generator HandshakeContextGenerator) {
	c.generator = generator
}

func (c *HandshakeGeneratorConnector) NewHandshakeContext() *HandshakeContext {
	return c.generator.NewHandshakeContext()
}
