package assets

import _ "embed"

//go:embed omnidesk.desktop
var DesktopEntry []byte

//go:embed omnidesk.service
var SystemdService []byte

//go:embed omnidesk.png
var IconPNG []byte

//go:embed omnidesk.svg
var IconSVG []byte

//go:embed omnidesk.ico
var IconICO []byte
