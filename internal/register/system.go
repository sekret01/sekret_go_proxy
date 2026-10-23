package register

import (
	_ "github.com/sekret01/sekret_go_proxy/internal/auths/composite_handshake"
	_ "github.com/sekret01/sekret_go_proxy/internal/auths/default"
	_ "github.com/sekret01/sekret_go_proxy/internal/dispatchers/default"
	_ "github.com/sekret01/sekret_go_proxy/internal/encryptors/default"
	_ "github.com/sekret01/sekret_go_proxy/internal/framers/default"
	_ "github.com/sekret01/sekret_go_proxy/internal/handshake/ipstep"
	_ "github.com/sekret01/sekret_go_proxy/internal/proxy/packets/linux"
	_ "github.com/sekret01/sekret_go_proxy/internal/transports/tcp"
)
