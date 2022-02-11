package main

import (
	"encoding/json"
	"math/rand"
	"net/http"

	"ec2/contratos"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/events"
	"github.com/segmentio/ksuid"
)

const (
	pErrEsc = 10 // Porciento aproximado de errores de "escaneo" que introducimos en las lecturas generadas, para test
	pErrSum = 10 // Porciento aproximado de errores de suma de verificación que introducimos en las lecturas generadas, para test
)

// Patrones con errores de "escaneo" usados para introducir errores en las lecturas, para test (pueden agregarse más)
// Aqui los 3 definidos:
// __   _   _
// | |  _| |_|
// |_| |_|   |

var errPatrones = [...]string{
	"__ | ||_|",
	" _  _||_|",
	" _ |_|  |",
}

func main() {
	lambda.Start(generaPaquete)
}

// Genera y sirve un paquete que contiene de 0 a 500 lecturas aproximadamente
func generaPaquete(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	canLecturas := rand.Intn(500) // De 0 a 500 lecturas, la cantidad en cada llamada es aleatoria

	// No hacemos seed del generador de random para que los resultados sean reproducibles en cada test

	// Si se genera 0 responde HTTP 404 Not Found (no hay paquete disponible)
	if canLecturas == 0 {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusNotFound,
			Headers: map[string]string{
				"Content-Type": "text/html",
			},
			Body: http.StatusText(http.StatusNotFound),
		}, nil
	}

	// Creamos un paquete de acuerdo al contrato y generamos una Id única para él mediante la librería ksuid
	paquete := contratos.Paquete{}
	paquete.Id = ksuid.New().String()

	// Generamos las lecturas del paquete, de ellas el porciento fijado (pErrSum) tendrá errores de suma de verificación
	for i := 0; i < canLecturas+1; i++ {
		paquete.Contenido = append(paquete.Contenido, generaLecturaYError())
	}

	// Formateamos "pretty" la respuesta a enviar
	respuesta, err := json.MarshalIndent(paquete, "", "  ")
	if err != nil { // Esto no debe suceder, pero....
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers: map[string]string{
				"Content-Type": "text/html",
			},
			Body: http.StatusText(http.StatusInternalServerError),
		}, nil
	}
	// Escribimos el paquete y HTTP 200 OK
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(respuesta),
	}, nil
}

// Genera una lectura
func generaLecturaYError() []string {
	lectura := []string{"", "", ""}
	for {
		// Para cada dígito de la lectura (generamos 8 y calculamos el noveno para cumplir con la suma de verificación)
		sum := 0
		for i := 0; i < 8; i++ {
			digito := rand.Intn(10) // generamos un digito aleatorio
			sum += (9 - i) * digito // vamos calculando la suma de chequeo
			// Codificamos el digito de acuerdo al patron de escaneo definido en el contrato, generando sus tres lineas
			for j := 0; j < 3; j++ {
				lectura[j] += contratos.Patrones[digito][3*j : 3*j+3]
			}
		}
		// Calculamos el dígito que cumple la suma
		ultDigito := ((sum/11)+1)*11 - sum
		if ultDigito == 10 {
			// Esta combinación no tiene un digito que haga cumplir la suma, desecharla y recomenzar
			lectura = []string{"", "", ""}
			continue
		}
		if ultDigito == 11 { // Si el resultado es 11, con la suma de los ocho se cumple la suma de verificación
			ultDigito = 0
		}
		// De acuerdo al porciento fijado para errores de suma (pErrSum) cambiamos este último dígito por el siguiente
		// Esto provoca un error de suma de verificación.
		if pErrSum > rand.Intn(101) {
			ultDigito = (ultDigito + 1) % 10
		}
		// Codificar el ultimo digito de acuerdo a los  patrones del contrato
		for j := 0; j < 3; j++ {
			lectura[j] += contratos.Patrones[ultDigito][3*j : 3*j+3]
		}
		break
	}
	// Generamos un error de escaneo de acuerdo al porciento fijado (aproximadamente)
	if pErrEsc > rand.Intn(101) {
		canDig := rand.Intn(10) // Número de dígitos a cambiar (aleatorio)
		for j := 0; j < canDig; j++ {
			digCamb := rand.Intn(9)                // dígito a cambiar
			digErrE := rand.Intn(len(errPatrones)) // seleccionamos un patrón erróneo aleatorio
			// Cambiamos el dígito por el erróneo. Indirecto, hay que sustuir, en cada una de las tres string la lectura,
			// el substring del dígito correcto por el substring del carácte erróneo (tienen 3 caracteres)
			for k := 0; k < 3; k++ {
				nString := lectura[k][0:digCamb*3] + errPatrones[digErrE][k*3:k*3+3] + lectura[k][digCamb*3+3:]
				lectura[k] = nString // asignamos la lectura ya con el patrón erróneo
			}
		}
	}
	// Regresamos la lectura generada
	return []string{lectura[0], lectura[1], lectura[2]}
}
