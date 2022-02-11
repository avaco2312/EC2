package contratos

type Paquete struct {
	Id        string
	Contenido [][]string
}

// Patrone de escaneo de los dígitos del 0 al 9
var Patrones = [...]string{
	" _ | ||_|",
	"     |  |",
	" _  _||_ ",
	" _  _| _|",
	"   |_|  |",
	" _ |_  _|",
	" _ |_ |_|",
	" _   |  |",
	" _ |_||_|",
	" _ |_| _|",
}
