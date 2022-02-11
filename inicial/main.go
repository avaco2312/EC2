package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
)

// Definimos los patrones que identifican cada número (3 líneas por 3 caracteres = 9)
var patrones map[string]string = map[string]string{
	" _ | ||_|": "0",
	"     |  |": "1",
	" _  _||_ ": "2",
	" _  _| _|": "3",
	"   |_|  |": "4",
	" _ |_  _|": "5",
	" _ |_ |_|": "6",
	" _   |  |": "7",
	" _ |_||_|": "8",
	" _ |_| _|": "9",
}

func main() {
	// Leer el archivo y convertir el resultado a string
	bdatos, err := ioutil.ReadFile("entradas.txt")
	if err != nil {
		log.Fatal("Error leyendo el archivo")
	}
	datos := string(bdatos)

	// Validamos el paquete recibido
	paquete := strings.ReplaceAll(datos, "\n", "")  // Eliminamos los finales de línea \n
	paquete = strings.ReplaceAll(paquete, "\r", "") // Eliminamos los finales de línea \r (Windows)
	if len(paquete)%81 != 0 {                       // Cada lectura debe ser exactamente 81 caracteres, el paquete es múltiplo de 81
		log.Fatal("Paquete de paquete incorrecto: No es múltiplo de 81")
	}
	if !regexp.MustCompile(`^[|_ ]+$`).MatchString(paquete) { // Verificar sólo caracteres admitidos: "|", "_" " ó " "
		log.Fatal(`Paquete de paquete incorrecto: Contiene caracteres que no son "|", "_" o " "`)
	}

	// Cantidad de lecturas  en el paquete (cada una con 9 dígitos correctos o no, cada dígito 9 * 3 *3 bytes)
	canLecturas := len(paquete) / 81

	// Crear el archivo "paraeljefe.txt" con los resultados
	fileJefe, err := os.Create("paraeljefe.txt")
	if err != nil {
		log.Fatal("Error creando el archivo para el jefe")
	}
	defer fileJefe.Close()

	// Procesa cada "lectura"
	for i := 0; i < canLecturas*81; i += 81 { // Para cada "lectura"
		lectura := ""
		for j := 0; j < 27; j += 3 { // Para cada "dígito de la lectura"
			patron := paquete[i+j:i+j+3] + paquete[i+j+27:i+j+30] + paquete[i+j+54:i+j+57] // Formamos el "patrón" de un dígito
			numero, ok := patrones[patron]                                                 // Ver si el "patrón" obtenido es un dígito "bien escaneado" o no (los patrones válidos definidos en datos.go)
			if !ok {
				lectura += "?" // Patrón no corresponde a ninguno de los patrones válidos
			} else {
				lectura += numero // Agregamos el dígito a la lectura
			}
		}
		// Para cada lectura, validamos
		if strings.Contains(lectura, "?") {
			// Contiene dígitos no válidos
			fmt.Fprintf(fileJefe,"%s\n",lectura + " ILL")
		} else {
			// Calculamos y validamos la suma de verificación
			fmt.Fprintf(fileJefe,"%s",lectura)
			sum := 0
			for i, digito := range []byte(lectura) {
				sum += int((digito - 48)) * (9 - i)
			}
			if sum%11 == 0 {
				fmt.Fprintf(fileJefe,"%s\n"," OK") // suma de verificación OK
			} else {
				fmt.Fprintf(fileJefe,"%s\n"," ERR") // incorrecta
			}
		}
	}
}
