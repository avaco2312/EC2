## Una "prueba técnica": Servicio Golang REST API local, sobre Docker, gRPC, AWS Serverless y sobre Kubernetes en AWS EC2 

La motivación de este repositorio está en una "prueba técnica" pedida por un "reclutador TI". Esta prueba, comparada con otras vistas, me pareció bastante racional.

El problema planteado, aunque no demasiado complejo, permite realizar diferentes implementaciones interesantes. Las dividiremos en:

  - [El problema inicial](#el-problema-inicial)
  - [Servidor y cliente tipo "REST API"](#servidor-y-cliente-tipo-rest-api)
  - [Correr el servidor sobre Docker](#correr-el-servidor-sobre-docker)
  - [Servidor y cliente gRPC](#servidor-y-cliente-grpc)
  - [Servidor sobre AWS Serverless](#servidor-sobre-aws-serverless)
  - [Servidor en Kubernetes K3S sobre una instancia de AWS EC2](#servidor-en-kubernetes-k3s-sobre-una-instancia-de-aws-ec2)

#### El problema inicial

En realidad nos interesan las implementaciones mencionadas, más que el problema en sí, por lo que lo describimos y damos una solución inicial, incompleta, pero que nos sirva de base.

Transcribimos el problema tal como se plantea en la prueba técnica:

Lector OCR

Imagina que trabajas para un banco, que recientemente compró una máquina ingeniosa para ayudarlo a leer cartas y faxes enviados a las sucursales. La máquina escanea los documentos en papel y produce un archivo con una cantidad de entradas que se ven así:

```
    _  _     _  _  _  _  _ 
  | _| _||_||_ |_   ||_||_|
  ||_  _|  | _||_|  ||_| _|
```

Cada entrada se conforma de 4 líneas y cada línea tiene 27 caracteres. Las primeras 3 líneas de cada entrada contienen un número de cuenta escrito usando pipes y guiones bajos, y la cuarta línea está en blanco. Cada número de cuenta debe tener 9 dígitos, todos los cuales deben estar en el rango 0-9. Un archivo normal contiene alrededor de 500 entradas.

La primera tarea es escribir un programa que pueda tomar este archivo y convertirlo en números de cuenta reales.

Una vez hecho lo anterior, rápidamente te das cuenta de que la ingeniosa máquina no es infalible. A veces comete errores en su escaneo. En consecuencia, el siguiente paso es validar que los números que lee son, de hecho, números de cuenta válidos. Se sabe que un número de cuenta válido tiene una suma de verificación válida y esto se puede calcular de la siguiente manera:

| número de cuenta: | 3   | 4   | 5   | 8   | 8   | 2   | 8   | 6   | 5   |
| ----------------- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| position name:    | d9  | d8  | d7  | d6  | d5  | d4  | d3  | d2  | d1  |

checksum calculation:

```
(1*d1 + 2*d2 + 3*d3 + … + 9*d9) mod 11 = 0

(1*5 + 2*6 + 3*8 + 4*2 + 5*8 + 6*8 + 7*5 + 8*4 + 9*3) = 231
231 mod 11 = 0
```

Se necesita escribir un programa que calcule la suma de verificación para un número de cuenta determinado e identifique si se trata de un número de cuenta válido.

Tu jefe está ansioso por ver sus resultados. Él te pide que escribas un archivo de los hallazgos, con una línea para cada número de cuenta, en este formato:

```
457508000 OK
664371495 ERR
86110??36 ILL
```

Es decir, el archivo tiene un número de cuenta por fila. Si algunos caracteres son ilegibles, se reemplazan por un '?'. En el caso de que alguna suma de verificación sea incorrecta o existe algún número ilegible, este estado se indica en una segunda columna.

Casos de prueba sugeridos:

```
 _  _  _  _  _  _  _  _
| || || || || || || ||_  |
|_||_||_||_||_||_||_| _| |
=> 000000051 OK

    _  _  _  _  _  _     _
|_||_|| || ||_   |  |  | _
  | _||_||_||_|  |  |  | _|
=> 49006771? ILL

    _  _     _  _  _  _   
  | _| _||_||_ |_   ||_||_|
  ||_  _|  | _||_|  ||_|  |
=> 123456784 ERR
```

Hasta aquí el planteamiento de la prueba.

Para resolverlo escribimos un solo programa, que contemple los casos planteados. El código en el subdirectorio "inicial".

Leemos el archivo con las lecturas de prueba, "entradas.txt", en una sola cadena de bytes que convertimos a string:

```
  bdatos, err := ioutil.ReadFile("entradas.txt")
  ...
  datos := string(bdatos)
```

El string leído contiene CR/LF que eliminamos:

```
  	paquete := strings.ReplaceAll(datos, "\n", "")  // Eliminamos los finales de línea \n
	paquete = strings.ReplaceAll(paquete, "\r", "") // Eliminamos los finales de línea \r (Windows)
```

Cada lectura contiene 9 dígitos y cada dígito se forma con tres líneas de tres caracteres. La longitud de una lectura es 9x3x3, 81 caracteres. El tamaño de un paquete de lecturas correcto es múltiplo de 81 caracteres. Chequeamos esto y también que sólo contenga los caracteres permitidos:

```
	if len(paquete)%81 != 0 {                       // Cada lectura debe ser exactamente 81 caracteres, el paquete es múltiplo de 81
		log.Fatal("Paquete de paquete incorrecto: No es múltiplo de 81")
	}
	if !regexp.MustCompile(`^[|_ ]+$`).MatchString(paquete) { // Verificar sólo caracteres admitidos: "|", "_" " ó " "
		log.Fatal(`Paquete de paquete incorrecto: Contiene caracteres que no son "|", "_" o " "`)
	}
```

Esta validación es burda. En realidad podríamos leer y validar cada lectura individual, en lugar del paquete como un todo, desechando sólo las lecturas incorrectas, etc. Pero para nuestro propósito basta. O si queremos quedar bien con los examinadores, esta es una solución "inicial" que continuaremos "refinando"...

En la forma que lo hemos hecho, el paquete es una "gran" cadena cuyo tamaño es un múltiplo de 81. Cada grupo de 81 caracteres es una lectura. Y en cada lectura, los primeros 27 caracteres son la primera línea de los 9 dígitos, los siguientes 27 la segunda línea y los 27 finales, la tercera línea. Así que tenemos que recorrer la "gran" cadena de esta forma, lectura a lectura y construyendo cada dígito de cada lectura. Esto lo hacemos con los ciclos:

```
	for i := 0; i < canLecturas*81; i += 81 { // Para cada "lectura"
		lectura := ""
		for j := 0; j < 27; j += 3 { // Para cada "dígito de la lectura"
			patron := paquete[i+j:i+j+3] + paquete[i+j+27:i+j+30] + paquete[i+j+54:i+j+57] // Formamos el "patrón" de un dígito

```

El ciclo i recorre cada lectura del paquete. El ciclo j recorre cada dígito de una lectura, formando un patron del dígito.

El patrón de un dígito será una cadena con los tres caracteres de la primera línea, más los tres de la segunda, más los tres de la tercera. Teniendo este patrón, podemos compararlo con un mapa de patrones correctos, que hemos creado anteriormente:

```
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
```

Si estos patrones los dividimos en grupos de a tres y los ponemos uno debajo del otro, en tres líneas, veremos que corresponden a los dígitos correctamente escaneados.

Luego, vemos si el patrón formado corresponde, o no, a uno de ellos:

```
			numero, ok := patrones[patron]  
```

Si corresponde, agregamos el dígito correspondiente a la lectura. Si no, es un patrón erróneo e incorporamos un "?":

```
			if !ok {
				lectura += "?" // Patrón no corresponde a ninguno de los patrones válidos
			} else {
				lectura += numero // Agregamos el dígito a la lectura
			}
```

Al terminar el ciclo j ya tenemos una lectura completa. Pasamos a validar: si contiene "?" escribimos la lectura con el estado ILL (previamente hemos creado el archivo "paraeljefe.txt", donde vamos escribiendo cada lectura):

```
		if strings.Contains(lectura, "?") {
			// Contiene dígitos no válidos
			fmt.Fprintf(fileJefe,"%s\n",lectura + " ILL")
```

Si la lectura tiene todos los dígitos correctos, la escribimos en el fichero y calculamos la suma de chequeo. Si la suma es divisible por 11 escribimos OK, en caso contrario ERR:

```
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
```

Y al final del ciclo i tendremos el archivo "paraeljefe.txt". Misión cumplida (o así lo creemos, esperar por la nota de los examinadores).

#### Servidor y cliente tipo "REST API"

Como la prueba trataba sobre un puesto de desarrollo backend, el examinador pidió preparar una solución "REST API". El planteamiento es vago, pero nos da la posibilidad de imaginar e implementar lo que queramos.

Bueno, tendremos un servidor "REST API", que suministra "paquetes de lecturas", y un cliente que los consume y produce los archivos correspondientes para el jefe. Los archivos de código en los directorios "servidor" y "cliente".

El servidor escucha en el puerto 9000 local y responde en el punto de entrada "/paquete" con el método "GET". A cada llamada devuelve un paquete que contiene un número aleatorio de lecturas (hasta 500). Queremos que la respuesta sea "más bonita y más REST", por eso pensamos en el siguiente formato JSON:

```
{
  "Id": "24vqPlCxsmuVv6OkMFUSspzJb7Y",
  "Contenido": [
    [
      " _  _  _     _  _  _  _    ",
      "  |  ||_|  ||_||_ | ||_ |_|",
      "  |  | _|  ||_| _||_||_|  |"
    ],
    [
      "    _  _  _        _  _  _ ",
      "  | _||_||_||_|  ||_   ||_|",
      "  ||_  _||_|  |  | _|  | _|"
    ],
	...
    [
      " _  _  _  _  _     _     _ ",
      " _|| |  |  | _||_||_|  |  |",
      " _||_|  |  ||_   ||_|  |  |"
    ]
  ]
}	
```

Agregamos una Id única para cada paquete (que identificará también los archivos "para el jefe"). Y si consultamos el servicio en un navegador o mediante curl, postman, etc., el resultado es "legible y bonito". (Esto es sólo una prueba, en el caso real posiblemente sea mejor ilegible y... encriptado).

Como en todo servicio que se respete, al haber un cliente y un servidor necesitamos establecer entre ellos un contrato, lo formalizamos en el directorio "contratos". Allí está la estructura de Go que les permitirá serializar-deserializar la información que se intercambia, en este caso el marshall-unmarshall JSON. Incluimos también los patrones válidos del escaner, permitiendo con ello poder cambiar algo esos patrones, sin afectar el servidor y el cliente (esta es la idea, realmente no lo hemos hecho general y nuestros cambios estarían limitados a "escaneos" de tres líneas y tres columnas)

```
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
```

No entraremos en demasiados detalles. La lógica del servidor es:

- Generar aleatoriamente la cantidad de lecturas que contendrá el paquete. En el "extraño" caso que sea 0, el servidor responderá 404 NOT FOUND, indicando que no hay paquete disponible.
- Generamos una Id única para el paquete, utilizando la librería "ksuid"
- Un ciclo, i, para generar las lecturas, llamando a la función "generaLecturaYError". Cada lectura se va agregando a la estructura de Go que materializa el paquete.
- Al finalizar el ciclo hacemos el marshal del paquete a JSON ("bonito, con indent") y escribimos el paquete. 

La función "generaLecturaYError" genera una lectura. Su lógica es:

- Genera aleatoriamente cada dígito, va agregando a la lectura los patrones de escaneo correspondientes y va acumulando la suma de verificación.
- Genera aleatoriamente sólo los primeros 8 dígitos. El noveno se calcula para que cumpla con que la suma de verificación sea divisible por 11.
- En este cálculo del último dígito hay un detalle: si el dígito está entre 0 y 9 se agrega. Si es 11, lo cambiamos a 0 y se cumple la verificación. Pero si el dígito es 10 implica que la combinación de 8 dígitos generada aleatoriamente no se puede completar de forma que cumpla la verificación, sólo queda desecharla y probar a generar de nuevo (instrucción continue)
- Queremos probar nuestro cliente, así que modificamos algunas lecturas para que sean erróneas (de acuerdo, aproximadamente, a unos porcientos que prefijamos, de errores de suma de verificación y de errores de escaneo)
- Para introducir un error de suma de verificación basta con sumar 1 al dígito final.
- Para los errores de escaneo, seleccionamos aleatoriamente algunos dígitos de la lectura y los sustituimos por patrones predefinidos, que sabemos son incorrectos.

Tampoco nos detendremos en el cliente, que es muy similar en lógica al programa "inicial". En este caso hace una petición http al servicio en el puerto 9000, recibe o no un paquete de lecturas, usa el contrato para deserializar en una estructura Go, procesa el paquete y escribe el resultado.

Lo que en este caso el fichero para el jefe se identificará también con la Id del paquete. Si volvemos a correr el cliente, obtenemos otro paquete de lecturas y el resultado se escribe en otro archivo.

Por ejemplo:

```
paraeljefe-24tfoCSE8Rp5Kj4OTQ6jsBWo3Ga.txt
paraeljefe-24vygIpnaXhfaoOnuz6OnmwNDDD.txt
```

#### Correr el servidor sobre Docker

Bueno, ya que llegamos aquí, podemos también correr el servidor sobre Docker. Esto también nos servirá más adelante.

Generamos el programa Linux ejecutable. Nos ubicamos en el directorio servidor y:

```
set GOOS=linux 
go build .
```

Creamos la imagen con el Dockerfile, también ubicado en el directorio "servidor":

```
FROM scratch
COPY servidor /
EXPOSE 9000
CMD ["/servidor"]
```

Para crear la imagen y posteriormente ejecutarla, damos los comandos siguientes (detener previamente el servidor local, de estar aún ejecutándose, para que deje libre el puerto 9000 local):

```
docker image build -t avaco/paquete .
docker run -d --name paquete -p 9000:9000 avaco/paquete
```
Si ejecutamos el cliente el resultado es el mismo que si corremos el servidor local.

El nombre dado a la image permitió hacer fácilmente push al repositorio público "avaco" en Docker Hub, donde está disponible. Esto también lo utilizaremos más adelante.

#### Servidor y cliente gRPC

Bueno, JSON y REST API están bien. Pero ¿por qué no gRPC? Montemos nuestro servidor y cliente sobre gRPC, el código en los directorios "protos", "servidor-grpc", "cliente-grpc" y "contract-grpc".

gRPC se basa en los proto buffers, creados por Google, que son una forma de formalizar la serialización-deserialización y el intercambio de mensajes en un formato binario muy eficiente. La definición se hace en un archivo "proto" que después se compila para el lenguaje que vamos a utilizar, en este caso Go, usando el compilador "protoc". Nuestro contrato ,inicial, es ahora "contratos.proto":

```
syntax = "proto3";
package contract_grpc;
option go_package = "./contract_grpc";

message Paquete {
	string id = 1; 
	repeated Lectura lectura = 2; 
}
message Lectura {
	repeated Linea linea = 1; 
}

message Linea{
    repeated string parte = 1;
}

message Empty {}

service GetPaquete {
    rpc LeePaquete(Empty) returns (Paquete) {}
}
```

El contrato entre servidor y cliente será de inicio este "proto". Definimos un mensaje tipo Paquete que se compone de una id y un conjunto de Lectura(s). Cada Lectura(s) es a su vez un conjunto de Linea(s) (en realidad serán 3 Linea(s)). También definimos un servicio GetPaquete que formaliza una llamada a un procedimiento remoto, LeePaquete. Este procedimiento no necesita parámetros (Empty) y retorna un mensaje de tipo Paquete.

La lógica es que nuestro cliente, a través de este procedimiento, pueda solicitar un paquete a otro equipo, el del servidor, y lo recibe mediante un mensaje de tipo Paquete. Claro, para nuestra prueba lo más probable es que ambos, cliente y servidor, estén en el mismo equipo.

El proto se compila para usarlo en Go. Obtenemos los archivos "contratos_grpc.pb.go" y "contratos.pb.go". Entre ambos conforman una librería Go, "contract_grpc", que podemos importar y utilizar en nuestro cliente y servidor gRPC.

La librería "contract_grpc" define el contrato a usar entre cliente y servidor y también la forma de llamada al procedimiento remoto. Genera estructuras Go para los mensajes. Para el servicio "GetPaquete" genera una versión destinada al servidor, "GetPaqueteServer", y una para el cliente, "GetPaqueteClient".

Para implementar como tal el servidor y el cliente gRPC utilizamos la librería "grpc", que nos permite crearlos y utilizar las estructuras creadas a partir del proto.

Sin muchos detalles (son extensos y hay amplia literatura sobre el tema), veamos el servidor:

```
listener, err := net.Listen("tcp", ":8090")
...
	obtienePaquete := sPaquete{}

	grpcServer := grpc.NewServer()
	contract_grpc.RegisterGetPaqueteServer(grpcServer, &obtienePaquete)
	reflection.Register(grpcServer)

	if err := grpcServer.Serve(listener); err != nil {
... 
type sPaquete struct {
	contract_grpc.UnimplementedGetPaqueteServer
}      
```

- Primero se crea un "listener" para escuchar en el puerto TCP 8090.
- Se crea un servidor gRPC "vacío".
- Definimos un tipo de estructura, sPaquete, que será el handler para atender las peticiones. Debe cumplir con la interfase del servicio "GetPaqueteServer", esto es, debemos implemetar un método "LeePaquete". Creamos una instancia de esa estructura, "obtienePaquete".
- Usando el paquete "contract_grpc" vinculamos el servidor gRPC con esa instancia "obtienePaquete", registrándolos como un "GetPaqueteServer". Esto es, al recibir una solicitud de un cliente, el servidor gRPC usará como handler a "obtienePaquete", este usará el método "LeePaquete" para satisfacerla.
- Se registra el servicio "reflection" en el servidor gRPC (un requisito de la librería gRPC).
- El servidor gRPC comienza a recibir peticiones en el puerto definido por el "listener". Para cada petición recibida lanzará una goroutine que ejecuta el método "LeePaquete" del handler y devuelve la respuesta, en este caso un mensaje de tipo Paquete, indicando además si ocurrió o no error.
- Al escribir el método "LeePaquete" debemos usar como contrato las definiciones del proto compilado (que se corresponden a las del proto inicial). Para el mensaje de entrada usamos contract_grpc.Empty. Para el mensaje de respuesta, contract_grpc.Paquete. En este método "LeePaquete" usamos el mismo esquema visto en los puntos anteriores para generar el paquete, pero usamos durante el procesamiento las estructuras proto generadas.
- Debemos comentar: por un problema de compatibilidad con versiones anteriores de gRPC, la estructura sPaquete no sólo debe cumplir con la interfase "GetPaqueteServer" y tener el método "LeePaquete", si no también debe cumplir con otras interfases contenidas en "UnimplementedGetPaqueteServer". De ahí incorporar este tipo en su definición, amablemente suministrado por el paquete "contract_grpc".

Con todo esto tendremos al servidor escuchando en el puerto 8090 para recibir peticiones al servicio "GetPaquete", las que invocan el procedimiento remoto "LeePaquete". 

Veamos el cliente:

```
	conn, err := grpc.Dial(":8090", grpc.WithInsecure())
...
	c := contract_grpc.NewGetPaqueteClient(conn)

	paquete, err := c.LeePaquete(context.Background(), &contract_grpc.Empty{})
...
```

- Creamos una conexión TCP cliente con el host y el puerto del servidor (en este caso la máquina local y el puerto 8090)
- Creamos una instancia, "c", de "GetPaqueteClient", la estructura definida en "contract_grpc" que representa el lado cliente de "GetPaquete" (contiene un método "LeePaquete").
- Usamos el método "LeePaquete" que posee "c" para solicitar un paquete al servidor. Pasamos el mensaje "Empty" y recibimos el resultado "paquete" que es un mensaje de tipo "Paquete" (en realidad *contract_grpc.Paquete).
- "paquete" contiene el paquete recibido, lo procesamos.

Ejecutamos el servidor. Cada vez que ejecutemos el cliente obtendremos un nuevo paquete para el jefe.

#### Servidor sobre AWS Serverless

¿Habremos impresionado a los examinadores? No... falta la nube. Así que pongamos nuestro servidor REST API en la nube de AWS.

Usaremos heramientas "serverless", esto es, poner "nuestra solución" en la nube sin necesidad de crear un servidor (hardware y software), sea físico o en la nube. Y podemos hacerlo gratis (durante un año), creando una cuenta gratis en AWS.

Las herramientas adecuadas (mínimas) son las funciones Lambda y el API Gateway Service. Las funciones Lambda es un servicio que permite definir pequeñas funciones de corta duración, que se ejecutan ante una petición o evento. API Gateway permite, entre otras cosas, crear un punto de entrada al que se puede solicitar una petición tipo REST, llamar una función Lambda que la atienda y devolver la respuesta al peticionario.

Así que crearemos un REST API de un solo punto de entrada GET, mediante API Gateway, que llamará a una función Lambda, que también crearemos. Esta procesará la petición (crear un paquete de lecturas), devolverá el resultado a API Gateway y este a nosotros.

Este es el servidor. El cliente es el mismo que está en el directorio "clientes", lo que debemos cambiar la dirección de llamada http, de nuestra máquina local puerto 9000 a la dirección que genere API Gateway para nuestra REST API.

Otra vez la extensión impide detenernos mucho. Pero en este caso es fácil: AWS ofrece magníficos tutoriales sobre API Gateway y Lambda. Son de corta duración, cubrwen lo que necesitaremos y... valen la pena.

Empezaremos con la función Lambda, el código en "servidor-aws":

```
func main() {
	lambda.Start(generaPaquete)
}

// Genera y sirve un paquete que contiene de 0 a 500 lecturas aproximadamente
func generaPaquete(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
...
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(respuesta),
	}, nil
}
```

- Toda función Lambda escrita en Go define el punto de entrada main de forma que redirija a la función que hace realmente el trabajo, en este caso "generaPaquete"
- Las funciones Lambda se activan ante la ocurrencia de diversos eventos. En nuestro caso, una llamada de API Gateway (que debemos definir que usa el modo "lambda proxy", más sobre esto posteriormente). 
- Cuando es llamada así, la función recibe unos parámetros de entrada, y debe reponder con unos parámetros de salida, estrictamente definidos. En este caso son del tipo APIGatewayProxyRequest y APIGatewayProxyResponse. 
- APIGatewayProxyRequest contiene múltiples parámetros de entrada, entre ellos, los parámetros recibidos de la llamada REST hecha a API Gateway (ip del cliente, path y método, query, body, etc.) En nuestro caso la petición REST a GET /paquete no contiene ningún parámetro de interés.
- APIGatewayProxyResponse nos permite devolver nuestra respuesta, en formato HTTP (StatusCode, Headers y el Body de la respuesta)
- Nuestro servidor recibe la petición, genera un paquete de lecturas, conforma y retorna la respuesta.
- Hay que compilar nuestra función lambda en formato linux, después hay que encapsularla en un archivo zip y subirla a AWS. Hay otras formas pero la más sencilla es utilizar el panel del servicio Lambda y seguir las instrucciones y tutoriales (hay secciones generales sobre funciones Lambda y una, muy útil, de Lambda usando Go)

Queda crear nuestra REST API con API Gateway. Igual, seguir el tutorial y unos detalles:

- Crearemos un solo recurso "/paquetes" para el que definiremos el método GET sin parámetros. También es bueno decir que si a la opción "activar CORS".
- En la implementación decimos que llamará a una función lambda, escogemos la que recién creamos (yo le puse el nombre "paquete"). Debemos además especificar que use "lambda proxy", para que los parámetros sean suministrados en el formato que usamos en nuestro programa (hay otros formatos de parámetros disponibles)
- La llamada a la API se puede probar dentro de API Gateway. Después que veamos que funciona, debemos generar la API, especificando un "stage", en mi caso usé "producción". Esto nos da la dirección a la que podemos llamar la API, para mí:

```
https://oy7rgdejtk.execute-api.us-west-2.amazonaws.com/produccion
```
y en el cliente sustituimos la llamada local por:

```
https://oy7rgdejtk.execute-api.us-west-2.amazonaws.com/produccion/paquete
```

Como esta es una petición GET sin parámetros, podemos verla también en el navegador. Por cierto, esta dirección está activa y pueden probarla, favor no hacerlo más de los miles de veces que ofrece la cuenta gratis de AWS :=)

#### Servidor en Kubernetes K3S sobre una instancia de AWS EC2

No hemos terminado. Intentemos dejar a los examinadores "patas arriba". Y esto es... Kubernetes. Hagamos el "deploy" de nuestro servidor en Kubernetes.

Kubernetes permite el funcionamiento de sistemas de aplicaciones complejos de forma eficiente, escalable, resiliente, autoregulable, etc., etc. Y además ¡está de moda!

Podríamos utilizar una distribución de Kubernetes en nuestra máquina local (kind y k3d son ideales para ello). Pero esto no sería demasiado impresionante, así que propongamos hacerlo en la nube de AWS. Los pasos iniciales serían:

- Crear una instancia de una máquina virtual EC2 con Amazon Linux 2. Podemos hacerlo con la cuenta gratis de AWS, si usamos una máquina poco potente, por ejemplo, tipo t2.micro.
- Aquí también seguir el tutorial de AWS sobre creación de instancias EC2. Los parámetros son todos los "default", excepto para la seguridad de la red. Debemos crear una regla de entrada que permita todo el tráfico TCP al puerto 32210, desde cualquier origen. Esto en un caso real no es seguro, pero en nuestro ejemplo lo necesitaremos.
- No olvidar generar y guardar en sitio seguro el certificado de seguridad de la instancia (archivo .pem) que genera AWS. Lo necesitaremos para conectarnos con la máquina virtual desde otro equipo.
- Lanzamos la instancia EC2 y nos conectamos a ella. Podemos hacerlo desde la propia consola de EC2, usando el navegador como terminal. O podemos establecer una conexión desde nuestro equipo usando SSH y el certificado. Todo esto desde una terminal linux (puede ser nativo o mediante Windows WSL). Y claro, podemos otra vez referirnos a las indicaciones y tutoriales de AWS, que nos guíen paso a paso.
- Una vez conectados, en el shell de la instancia EC2 damos los siguientes comandos:

```
[ec2-user@ip-172-31-3-146 ~]$ sudo yum update
[ec2-user@ip-172-31-3-146 ~]$ curl -sfL https://get.k3s.io | sh -s - --write-kubeconfig-mode 644
```

- El primer comando actualiza el sistema linux de nuestra instancia.
- El segundo instala una distribución de Kubernetes, K3S, que está pensada para configuraciones limitadas, como la nuestra.
- En ambos casos la salida por la consola es extensa y demora algo. Al terminar tendremos un cluster Kubernetes K3S, listo para ser usado (Kubernetes se identifica como K8S, esta "es solo K3S" :=)

Y ¿qué es un cluster Kubernetes? Antes de responder, comprobemos que realmente lo tenemos listo. En la consola tecleamos:

```
[ec2-user@ip-172-31-3-146 ~]$ kubectl get nodes
NAME                                         STATUS   ROLES                  AGE   VERSION
ip-172-31-3-146.us-west-2.compute.internal   Ready    control-plane,master   13m   v1.22.6+k3s1
```
Si obtuvimos una respuesta similar tenemos un cluster, con un nodo, listo para ser usado.

Kubernetes es amplio y complejo, tratemos de simplificar un poco (bastante). Pensemos que un cluster Kubernetes es un espacio aislado donde se ejecutan aplicaciones. Puede estar constituido por uno o más nodos, y los nodos pueden estar en una o diferentes máquinas, reales o virtuales. Todos los nodos están interconectados mediante redes, reales o virtuales.

Hay dos tipos de nodos. Siempre habrá uno o más nodos "master" que cumplen la función de dirigir y coordinar todo el trabajo, de ellos y de otros nodos, además de poder ejecutar tareas de procesamiento de las aplicaciones. Además podemos tener cero o más nodos "workers", que sólo contienen tareas de las aplicaciones que corren en el cluster. En nuestro caso tendremos un sólo nodo, que servirá de master y además correrá las tareas de nuestra aplicación, haciendo también de worker.

La unidad de ejecución en un nodo es el "pod". Pensemos que el pod es una mini-máquina donde se ejecutan una o más aplicaciones encapsuladas cada una en un contenedor. En nuestro caso será sencillo: el servidor de paquetes de lecturas estará encapsulado en un contenedor que se ejecuta en un único pod.

Podemos crear directamente el pod, pero la mejor práctica es hacerlo mediante un "deployment". Esta es una estructura de control que permite crear uno o más pods del mismo tipo. Con varias ventajas: una vez creados, los monitorea y si alguno falla lo vuelve a relanzar automáticamente.

También permite crear a la vez un número de réplicas de un pod. Así si queremos podemos tener más de un generador de paquetes corriendo a la vez, cada uno en un pod diferente, todos creado y manejados por el deployment. También variar el número de réplicas en ejecución, aumentar o disminuir, lo que permite escalar o desescalar la disponibilidad de los servicios, de acuerdo a la demanda.

Los pods que están corriendo están aislados unos de otros y con respecto al exterior. Para accesar a sus puntos de entrada necesitamos los "servicios". Los servicios son estructuras que "subordinan" uno o más pods. Las peticiones que llegan a la dirección ip del servicio se transmiten a uno de los pods subordinados. Los servicios son visibles a los otros elementos que se ejecutan en el cluster.

Luego, para nuestro servidor crearemos un pod, usando un deployment. Especificamos la imagen que contendrá el contenedor que correrá dentro del pod, que será la imagen que creamos anteriormente, disponible de manera pública en el repositorio avaco/paquete. También señalamos el puerto donde escucha el pod, que será el puerto 9000.

Creamos un servicio, que asociará a él el pod creado. En realidad lo haremos al revés, primero el pod y después el servicio, el orden no importa, siempre que se cree algo, se chequea si hay otros elementos ya existentes que deban ser asociados. Para seleccionar sus pods, el servicio utiliza un "selector": siempre que el pod tenga una "etiqueta" coincidente con ese selector, el pod se asocia al servicio.

Una forma de crear los elementos en Kubernetes es declarativa, describiéndolos en un fichero ".yaml". Veamos algo del que usaremos, paquete.yaml en el directorio kubernetes:

```
apiVersion: apps/v1
kind: Deployment
metadata:
  name:  paquete
...
  selector:
    matchLabels:
      app: paquete
  replicas: 1      
...
    spec:
      containers:
      - name:  paquete
        image:  avaco/paquete:latest
...
        - containerPort:  9000
```

La sintaxis es compleja y tersa, incluso el correcto indentado es un requisito. En esta primera parte creamos el deployment y con ello los pods correspondientes (en realidad será un sólo pod)

- El nombre del deployment es paquete. Cada pod que cree recibirá como nombre "paquete" más una "cola" aleatoria, por ejemplo, paquete-75bb5556d8-rwgbh. El nombre de cada pod (si hay más de uno) es único.
- Los pods que se creen se asociarán al servicio cuyo selector sea app: paquete
- Sólo se creará, y se controlará, un solo pod (replicas: 1)
- El pod se creará habilitando un contenedor, usando para ello la imagen paquete del repositorio avaco.
- Una vez creado el contenedor, se ejecuta el comando inicial definido en la imagen. El contenedor y el pod escuchan y reciben peticiones en el puerto 9000, en la dirección ip que se asigna al pod.

Como dijimos, la dirección ip asignada al pod no es visible a otros elementos del cluster, ni al exterior. Para llegar a él creamos un servicio:

```
apiVersion: v1
kind: Service
metadata:
  name: paquete
  namespace: default
spec:
  selector:
    app: paquete
  type: NodePort
...
  ports:
  - name: paquete
    protocol: TCP
    port: 9000
    targetPort: 9000
    nodePort: 32210  
```

- El servicio se llama paquete.
- Su selector es app: paquete, asocia el pod que creamos con el deployment anterior.
- El servicio escucha en la dirección ip que se le asigna, en el puerto 9000. Redirigirá las peticiones que le lleguen a uno de los pods asociados, usando la ip del pod y el puerto 9000.

Con ello las peticiones al servicio en el puerto 9000 llegan al pod creado, en el puerto 9000, y se genera y devuelve un paquete de lecturas.

La dirección de un servicio es visible a todos los elementos del cluster, en la dirección ip que se le asigna, en el puerto indicado. Pero no es visible al exterior del cluster.

Para accesar desde el exterior declaramos adicionalmente este servicio como de tipo NodePort, indicando se asigne el puerto 32210. El resultado es que además de su ip y puerto a nivel de cluster, se le asocia la dirección que tiene el nodo con el puerto pedido. Como las direcciones ip de los nodos son las de las correspondientes máquinas en que corren, si estas son visibles al exterior, nuestro servicio también lo será.

Para nuestro caso, el NodePort asocia el servicio a la dirección del nodo (y de la máquina EC2 en que este corre), en el puerto indicado, 32210. Como al crear la máquina EC2 hemos autorizado a todos los usuarios poder usar ese puerto, el servicio será accesible globalmente (si, en cualquier parte del mundo).

Claro, existen formas más eficientes y seguras de exponer nuestros servicios. Pero son más complejas y van más allá de un ejemplo. Y además, no están disponibles con una cuenta gratis de AWS.

Necesitamos tener el archivo "paquete.yaml" en el directorio local de la instancia EC2, para poder usarlo. Para llevarlo allí, desde nuestro equipo, se me ocurren varios procedimientos:

- Recrearlo usando un editor de texto en la instancia EC2. Tarea engorrosa y propensa a errores, más difícil si sólo contamos allí con editores "primitivos".
- Usar el servicio de almacenamiento S3 de AWS. Copiamos de nuestra máquina local a S3, usando el cliente de AWS y después desde S3 a la instancia EC2, usando la consola de la instancia. Los tutoriales correspondientes en AWS.
- Para mi la más sencilla, usar un cliente FTP como filezilla. Establecemos una sesión FTP con la instancia EC2, que debe ser a través de SSH. Para conectar necesitamos la ip pública de la instancia (visible en la consola de EC2) y el certificado pem. Una vez conectados podemos copiar del sistema de archivos local al de la instancia.

Para desplegar los elementos definidos en el archivo yaml utilizamos el comando kubectl, que es la inteface de comandos de Kubernetes:

```
[ec2-user@ip-172-31-3-146 ~]$ kubectl apply -f paquete.yaml
deployment.apps/paquete created
service/paquete created
```

Podemos comprobar si el deployment creó el pod y si este está listo:

```
[ec2-user@ip-172-31-3-146 ~]$ kubectl get pods
NAME                       READY   STATUS    RESTARTS   AGE
paquete-75bb5556d8-rwgbh   1/1     Running   0          5s
```

También si el servicio está listo y escuchando:

```
[ec2-user@ip-172-31-3-146 ~]$ kubectl get svc
NAME         TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)          AGE
kubernetes   ClusterIP   10.43.0.1      <none>        443/TCP          8m46s
paquete      NodePort    10.43.128.36   <none>        9000:32210/TCP   11s
```

Observen que la respuesta indica que el servicio paquete está activo y escucha en dos "niveles", a nivel de cluster en la ip asignada 10.43.128.36, puerto 9000. A nivel global, en la dirección de cada nodo (que es el de la de la máquina que corre el nodo) y el puerto 32210.

Luego podemos llamar a nuestro generador de paquetes y recibir la respuesta usando el cliente. La dirección ip asignada a la instancia EC2 se encuentra en la consola, en mi caso, 35.86.246.146. En el cliente cambiamos la dirección a llamar por:

http://35.86.246.146:32210/paquete

AWS también asigna un nombre DNS (un poco largo) que se resuelve a esa dirección ip. También aparece en la consola. La dirección de llamada usando ese nombre sería:

http://ec2-35-86-246-146.us-west-2.compute.amazonaws.com:32210/paquete

Últimos comentarios:

- Las direcciones asignadas a las instancias EC2 se mantienen mientras no se detenga la instancia. De detenerla y reniciarla las direcciones cambian (claro, cuando usamos un  perfil "sólo gratis")
- No intente cargar algo real a este cluster Kubernetes. La instancia EC2 gratis es poco potente y "explota" si pretendemos correr algo más pesado. Para pruebas más complejas es mejor una variante de Kubernetes local. O dinero y contratar en "grande" en la nube.
- No es lo mejor, para muchos, hacer un deploy "propio" de Kubernetes. Existen múltiples proveedores de servicios en la nube, incluyendo AWS, que ofertan "Kubernetes como servicio", ahorrando no sólo costos, si no también el gran esfuerzo en configuración, mantenimiento y optimización que requiere un cluster "propio".