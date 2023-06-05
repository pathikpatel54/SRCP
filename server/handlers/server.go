package server

import (
	"crypto/tls"
	"encoding/binary"
	"log"
	"net"
	"os"
	"scrp/models"
	"scrp/server/utils"
	"scrp/variables"
	"sync"
)

type State int

const (
	INIT State = iota
	AUTH_REQ_RECVD
	AUTHENTICATED
	PUBLIC_KEY_RECVD
	PUBLIC_KEY_SENT
	CHAT
	DISCONNECTING
	TERMINATED
)

type Server struct {
	Listener net.Listener
	Clients  map[string]*Client
	mutex    sync.Mutex
}

type Client struct {
	Username [32]byte
	Key      [512]byte
	Conn     net.Conn
	State    State
}

func NewServer() *Server {
	return &Server{
		Clients: make(map[string]*Client),
	}
}

func (s *Server) Listen(port string) {
	certFile := "cert.pem"
	keyFile := "key.pem"

	_, certInfo := os.Stat(certFile)
	_, keyInfo := os.Stat(keyFile)

	if os.IsNotExist(certInfo) || os.IsNotExist(keyInfo) {
		log.Println("Generating server certificates and key...")
		if err := utils.GenerateServerCert(certFile, keyFile); err != nil {
			log.Fatalf("Failed to generate server certificates and key: %v", err)
		}
		log.Println("Server certificates and key generated.")
	}

	cert, err := tls.LoadX509KeyPair("./server/"+certFile, "./server/"+keyFile)
	if err != nil {
		log.Fatalf("Failed to load server certificate: %v", err)
	}

	config := tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	s.Listener, err = tls.Listen("tcp", ":8080", &config)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	log.Println("Listening on port 8080...")

	for {
		conn, err := s.Listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		go s.HandleClient(conn)
	}
}

func (s *Server) HandleClient(conn net.Conn) {
	defer conn.Close()

	// Read and process messages from the client
	for {
		// Read the header
		var header models.Header
		err := binary.Read(conn, binary.BigEndian, &header)

		if err != nil {
			log.Printf("Failed to read header from client: %v", err)
			return
		}
		// Read the payload based on the message type
		switch header.Type {
		case variables.AuthRequest:
			var payload models.AuthRequestPayload
			err = binary.Read(conn, binary.BigEndian, &payload)
			if err != nil {
				log.Printf("Failed to read AUTH_REQUEST payload from client: %v", err)
				return
			}
			s.HandleAuthRequest(conn, payload)

		case variables.KeyExchange:
			var payload models.PublicKeyPayload
			err = binary.Read(conn, binary.BigEndian, &payload)
			if err != nil {
				log.Printf("Failed to read KEY_EXCHANGE payload from client: %v", err)
				return
			}

			s.HandleKeyExchange(conn, payload)
			// TODO: Handle public key exchange

		case variables.Message:
			var payload models.MessagePayload
			err = binary.Read(conn, binary.BigEndian, &payload)
			if err != nil {
				log.Printf("Failed to read MESSAGE payload from client: %v", err)
				return
			}
			s.HandleMessage(conn, payload)
			// TODO: Handle incoming message

		case variables.Disconnect:
			var payload models.DisconnectPayload
			err = binary.Read(conn, binary.BigEndian, &payload)
			if err != nil {
				log.Printf("Failed to read DISCONNECT payload from client: %v", err)
				return
			}

			s.HandleDisconnect(conn, payload)

		default:
			log.Printf("Unknown message type received from client: %d", header.Type)
			return
		}
	}
}
