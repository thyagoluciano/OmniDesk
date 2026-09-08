package assets

import _ "embed"

//go:embed crossover.desktop
var DesktopEntry []byte

//go:embed crossover.service
var SystemdService []byte

//go:embed crossover.png
var IconPNG []byte

//go:embed crossover.svg
var IconSVG []byte
