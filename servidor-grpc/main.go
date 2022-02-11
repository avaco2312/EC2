package main

import (
	"context"

	//"fmt"
	"log"
	"math/rand"
	"net"
	"ec2/contract_grpc"
	"ec2/contratos"

	"github.com/segmentio/ksuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
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
	// Servidor gRPC en :8080

	listener, err := net.Listen("tcp", ":8090")
	if err != nil {
		log.Fatal("Error de listener TCP")
	}

	obtienePaquete := sPaquete{}

	grpcServer := grpc.NewServer()
	contract_grpc.RegisterGetPaqueteServer(grpcServer, &obtienePaquete)
	reflection.Register(grpcServer)

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %s", err)
	}

}

type sPaquete struct {
	contract_grpc.UnimplementedGetPaqueteServer
}

func (s *sPaquete) LeePaquete(ctx context.Context, in *contract_grpc.Empty) (*contract_grpc.Paquete, error) {
	canLecturas := rand.Intn(500) // De 0 a 500 lecturas, la cantidad en cada llamada es aleatoria

	// No hacemos seed del generador de random para que los resultados sean reproducibles en cada test

	// Si se genera 0 responde HTTP 404 Not Found (no hay paquete disponible)
	if canLecturas == 0 {
		return &contract_grpc.Paquete{Id: "0"}, nil
	}

	// Creamos un paquete de acuerdo al contrato y generamos una Id única para él mediante la librería ksuid
	paquete := contract_grpc.Paquete{Id: ksuid.New().String()}

	// Generamos las lecturas del paquete, de ellas el porciento fijado (pErrSum) tendrá errores de suma de verificación
	var tempLectura []*contract_grpc.Lectura

	for i := 0; i < canLecturas+1; i++ {
		lin := generaLecturaYError()
		contlectura := &contract_grpc.Lectura{
			Linea: []*contract_grpc.Linea{
				{
					Parte: []string{lin[0], lin[1], lin[2]},
				},
			},
		}
		tempLectura = append(tempLectura, contlectura)
	}
	paquete.Lectura = tempLectura
	return &paquete, nil
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
