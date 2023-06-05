package handlers

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"scrp/models"
	"scrp/variables"
	"strconv"
	"sync"
)

type State int

const (
	INIT State = iota
	AUTH_REQ_SEND
	AUTHENTICATED
	PUBLIC_KEY_SENT
	PUBLIC_KEY_RECVD
	CHAT
	DISCONNECTING
	TERMINATED
)

type Client struct {
	Username        [32]byte
	Password        [32]byte
	OwnPublicKey    [512]byte
	OwnPrivateKey   *rsa.PrivateKey
	OtherPublicKeys map[string][512]byte
	Conn            net.Conn
	Mode            string
	State           State
	mutex           sync.Mutex
}

func NewClient(username string, password string) (*Client, error) {
	// Generate RSA keys
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("could not generate private key: %v", err)
	}

	// Convert public key to PKIX, ASN.1 DER form
	pubBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("could not marshal public key: %v", err)
	}

	// Ensure the public key byte array is the correct size
	var pubKeyArr [512]byte
	copy(pubKeyArr[:], pubBytes)

	return &Client{
		Username:        stringToByteArray32(username),
		Password:        stringToByteArray32(password),
		OwnPublicKey:    pubKeyArr,
		OwnPrivateKey:   privateKey,
		OtherPublicKeys: make(map[string][512]byte),
		State:           INIT,
	}, nil
}

func stringToByteArray32(s string) [32]byte {
	var byteArray [32]byte
	copy(byteArray[:], s)
	return byteArray
}

func (c *Client) SendAuthRequest() {
	header := models.Header{
		Version:  variables.Version,
		Type:     variables.AuthRequest,
		Length:   uint16(binary.Size(c.Username) + binary.Size(c.Password)),
		Sequence: 0, // Sequence number, update this as needed
	}

	payload := models.AuthRequestPayload{
		Username: c.Username,
		Password: c.Password,
	}

	// Write header and payload
	err := binary.Write(c.Conn, binary.BigEndian, &header)
	if err != nil {
		log.Printf("Failed to write header to server: %v", err)
		return
	}

	err = binary.Write(c.Conn, binary.BigEndian, &payload)
	if err != nil {
		log.Printf("Failed to write payload to server: %v", err)
	}
}

func (c *Client) HandleServerMessages() {
	go func() {
		for {
			// Read the header
			var header models.Header
			err := binary.Read(c.Conn, binary.BigEndian, &header)
			if err != nil {
				log.Printf("Failed to read header from server: %v", err)
				return
			}

			// Read the payload based on the message type
			switch header.Type {
			case variables.AuthResponse:
				// Handle AUTH_RESPONSE based on the current state
				switch c.State {
				case INIT:
					var payload models.AuthResponsePayload
					err = binary.Read(c.Conn, binary.BigEndian, &payload)
					if err != nil {
						log.Printf("Failed to read AUTH_RESPONSE payload from server: %v", err)
						return
					}
					if payload.Status == variables.AuthSuccess {
						// Authentication successful, transition to the next state
						c.State = AUTHENTICATED
						c.SendPublicKey()
						c.State = PUBLIC_KEY_SENT
					} else {
						log.Println("Authentication failed.")
						return
					}
				default:
					log.Printf("Received AUTH_RESPONSE in an unexpected state: %v", c.State)
					return
				}

			case variables.KeyExchange:
				// Handle KEY_EXCHANGE based on the current state
				switch c.State {
				case PUBLIC_KEY_SENT, CHAT, PUBLIC_KEY_RECVD:
					var payload models.PublicKeyPayload
					err = binary.Read(c.Conn, binary.BigEndian, &payload)
					if err != nil {
						log.Printf("Failed to read KEY_EXCHANGE payload from server: %v", err)
						return
					}

					// Handle other Client disconnect
					if c.State == PUBLIC_KEY_RECVD || c.State == CHAT {
						var checker [512]byte
						username := string(bytes.Trim(payload.Username[:], "\x00"))
						if payload.Key == checker {
							c.mutex.Lock()
							delete(c.OtherPublicKeys, username)
							c.mutex.Unlock()
							c.Mode = "Select"
							c.State = PUBLIC_KEY_RECVD
							c.DisplayParticipants()
						} else {
							// Add public key to map
							c.mutex.Lock()
							username := string(bytes.Trim(payload.Username[:], "\x00"))
							c.OtherPublicKeys[username] = payload.Key
							c.mutex.Unlock()

							// Transition to the next state
							c.State = PUBLIC_KEY_RECVD

							// Display the list of participants
							c.DisplayParticipants()
						}
					} else {
						// Add public key to map
						c.mutex.Lock()
						username := string(bytes.Trim(payload.Username[:], "\x00"))
						c.OtherPublicKeys[username] = payload.Key
						c.mutex.Unlock()

						// Transition to the next state
						c.State = PUBLIC_KEY_RECVD

						// Display the list of participants
						c.DisplayParticipants()
					}

				default:
					log.Printf("Received KEY_EXCHANGE in an unexpected state: %v", c.State)
					return
				}

			case variables.Message:
				// Handle MESSAGE based on the current state
				switch c.State {
				case CHAT, PUBLIC_KEY_RECVD:
					c.State = CHAT
					var payload models.MessagePayload
					err = binary.Read(c.Conn, binary.BigEndian, &payload)
					if err != nil {
						log.Printf("Failed to read MESSAGE payload from server: %v", err)
						return
					}

					// Decrypt the data using own private key
					decryptedData, err := c.DecryptData(bytes.Trim(payload.Data[:], "\x00")) // Implement DecryptData function
					if err != nil {
						log.Printf("Failed to decrypt payload: %v", err)
						return
					}

					// Show the message
					clearLine()
					fmt.Printf("%s: %s", payload.Sender, decryptedData)
					fmt.Printf("\nYour Message: ")

				default:
					log.Printf("Received MESSAGE in an unexpected state: %v", c.State)
					return
				}

			default:
				log.Printf("Unknown message type received from server: %d", header.Type)
				return
			}
		}
	}()
}

func (c *Client) DisplayParticipants() {
	if c.Mode == "Select" {
		clearScreen()
		fmt.Println("Participant List:")
		fmt.Println("=================")
		i := 1
		for username := range c.OtherPublicKeys {
			fmt.Printf("%d. %s\n", i, username)
			i++
		}
		fmt.Printf("=================\n\n")

		fmt.Printf("Enter participant number to chat: ")
	}
}

func (c *Client) StartMessagingUI() {

	scanner := bufio.NewScanner(os.Stdin)

	for {
		c.Mode = "Select"

		c.DisplayParticipants()

		scanner.Scan()
		input := scanner.Text()

		participantNumber, err := strconv.Atoi(input)

		if err != nil {
			fmt.Println("Invalid participant number. Please try again.")
			continue
		}

		i := 1
		var recipientUsername string
		for username := range c.OtherPublicKeys {
			if i == participantNumber {
				recipientUsername = username
				break
			}
			i++
		}

		if recipientUsername == "" {
			fmt.Println("Invalid participant number. Please try again.")
			continue
		}

		for {
			c.Mode = "Message"
			c.State = CHAT

			fmt.Print("Your Message: ")
			scanner.Scan()
			message := scanner.Text()

			if message == "" || c.State == PUBLIC_KEY_RECVD {
				break
			}

			// Encrypt the data using recipient's public key
			encryptedData, err := c.EncryptData([]byte(message), recipientUsername)
			if err != nil {
				log.Printf("Failed to encrypt message: %v", err)
				return
			}

			var data [2048]byte
			copy(data[:], encryptedData)
			if err != nil {
				log.Printf("Failed to write header to server: %v", err)
				return
			}

			header := models.Header{
				Version:  variables.Version,
				Type:     variables.Message,
				Length:   uint16(binary.Size(encryptedData)),
				Sequence: 0, // Sequence number, update this as needed
			}

			payload := models.MessagePayload{
				Timestamp: 0, // Update this with the current timestamp
				Sender:    c.Username,
				Recipient: [32]byte(stringToByteArray32(recipientUsername)),
				TextLen:   uint16(len(message)),
				Data:      data,
			}

			// Write header and payload
			err = binary.Write(c.Conn, binary.BigEndian, &header)
			if err != nil {
				log.Printf("Failed to write header to server: %v", err)
				return
			}

			err = binary.Write(c.Conn, binary.BigEndian, &payload)
			if err != nil {
				log.Printf("Failed to write payload to server: %v", err)
			}
		}
	}
}

func (c *Client) EncryptData(plaintext []byte, recipientUsername string) ([]byte, error) {
	publicKey, ok := c.OtherPublicKeys[recipientUsername]
	if !ok {
		return nil, fmt.Errorf("public key for user %s not found", recipientUsername)
	}
	pubInterface, err := x509.ParsePKIXPublicKey(bytes.Trim(publicKey[:], "\x00"))
	if err != nil {
		return nil, err
	}
	pub, ok := pubInterface.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("could not cast publicKey to rsa.PublicKey")
	}
	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, pub, plaintext)
	if err != nil {
		return nil, err
	}
	return encrypted, nil
}

func (c *Client) DecryptData(ciphertext []byte) ([]byte, error) {
	plaintext, err := rsa.DecryptPKCS1v15(rand.Reader, c.OwnPrivateKey, ciphertext)
	if err != nil {
		return nil, fmt.Errorf("could not decrypt ciphertext: %v", err)
	}
	return plaintext, nil
}

func (c *Client) SendPublicKey() {
	header := models.Header{
		Version:  variables.Version,
		Type:     variables.KeyExchange,
		Length:   uint16(binary.Size(c.OwnPublicKey)),
		Sequence: 0, // Sequence number, update this as needed
	}

	payload := models.PublicKeyPayload{
		Username: c.Username,
		Key:      c.OwnPublicKey,
	}
	// Write header and payload
	err := binary.Write(c.Conn, binary.BigEndian, &header)
	if err != nil {
		log.Printf("Failed to write header to server: %v", err)
		return
	}
	err = binary.Write(c.Conn, binary.BigEndian, &payload)
	if err != nil {
		log.Printf("Failed to write payload to server: %v", err)
	}
}

func (c *Client) SendDisconnectRequest() {
	header := models.Header{
		Version:  variables.Version,
		Type:     variables.Disconnect,
		Length:   uint16(0),
		Sequence: 0, // Sequence number, update this as needed
	}

	payload := models.DisconnectPayload{
		Reason: variables.UserRequest,
	}
	// Write header and payload
	err := binary.Write(c.Conn, binary.BigEndian, &header)
	if err != nil {
		log.Printf("Failed to write header to server: %v", err)
		return
	}
	err = binary.Write(c.Conn, binary.BigEndian, &payload)
	if err != nil {
		log.Printf("Failed to write payload to server: %v", err)
	}
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func clearLine() {
	fmt.Print("\033[2K\r")
}
