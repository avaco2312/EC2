package main

import (
	"fmt"
	"log"
	"os"
	"ec2/contract_grpc"
	"ec2/contratos"
	"regexp"
	"strings"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.Dial(":8090", grpc.WithInsecure())
	if err != nil {
		log.Fatal("Error de dial")
	}
	defer conn.Close()
	c := contract_grpc.NewGetPaqueteClient(conn)

	paquete, err := c.LeePaquete(context.Background(), &contract_grpc.Empty{})
	if err != nil {
		log.Fatal("Error de call grpc ", err)
	}
	if paquete.Id == "0" {
		log.Fatal("No había paquete disponible")
	}
	log.Printf("Recibidas %d lecturas", len(paquete.Lectura))
	// Creamos un mapa de los patrones y su correspondiente dígito, de acuerdo al contrato
	patrones := map[string]string{}
	for i, pat := range contratos.Patrones {
		patrones[pat] = fmt.Sprint(i)
	}

	// Crear el archivo "paraeljefe-(Id del paquete).txt" que contendrá las lecturas y su validación
	fileJefe, err := os.Create("paraeljefe-" + paquete.Id + ".txt")
	if err != nil {
		log.Fatal("Error creando el archivo para el jefe")
	}
	defer fileJefe.Close()

	for i, rawLectura := range paquete.Lectura { // Para cada lectura
		lectura := ""
		// Unimos las tres filas de la lectura en una sola
		for _, rawFila := range rawLectura.Linea {
			for _, rawParte := range rawFila.Parte {
				lectura += rawParte
			}
		}
		if len(lectura) != 81 { // La lectura debe tener este tamaño (9 dígitos x 3 filas x 3 caracteres)
			log.Printf("error en la lectura %d: tamaño incorrecto %d", i, len(lectura))
			continue // log y deschamos esta lectura
		}
		if !regexp.MustCompile(`^[|_ ]+$`).MatchString(lectura) { // Verificar sólo caracteres admitidos: "|", "_" " ó " "
			log.Printf(`error en la lectura %d: contiene caracteres que no son "|", "_" o " "`, i)
			continue // log y deschamos esta lectura
		}
		// Convertimos la lectura a formato "numérico" y validamos errores de escaneo
		nlectura := ""
		for j := 0; j < 27; j += 3 { // Para cada "dígito de la lectura"
			patron := lectura[j:j+3] + lectura[j+27:j+30] + lectura[j+54:j+57] // Formamos el "patrón" de un dígito
			numero, ok := patrones[patron]                                     // Ver si el "patrón" obtenido es un dígito "bien escaneado" o no
			if !ok {
				nlectura += "?" // Patrón no corresponde a ninguno de los patrones válidos
			} else {
				nlectura += numero // Agregamos el dígito a la lectura
			}
		}
		// Para cada lectura, validamos y si es "numérica" calculamos y validamos la suma de verificación
		if strings.Contains(nlectura, "?") {
			// Contiene dígitos no válidos
			fmt.Fprintf(fileJefe, "%s\n", nlectura+" ILL")
		} else {
			// Calculamos y validamos la suma de verificación
			fmt.Fprintf(fileJefe, "%s", nlectura)
			sum := 0
			for i, digito := range []byte(nlectura) {
				sum += int((digito - 48)) * (9 - i)
			}
			if sum%11 == 0 {
				fmt.Fprintf(fileJefe, "%s\n", " OK") // suma de verificación OK
			} else {
				fmt.Fprintf(fileJefe, "%s\n", " ERR") // incorrecta
			}
		}
	}

}
