package server

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"scrp/models"
	"scrp/variables"
)

func (s *Server) HandleAuthRequest(conn net.Conn, payload models.AuthRequestPayload) {
	// Assume that the username and password are correct
	// Retrieve the client's username from the payload
	username := string(bytes.Trim(payload.Username[:], "\x00"))

	log.Println("User Authenticated: ", username)

	// Perform authentication logic here (e.g., checking against stored credentials)
	authenticated := true // Replace with your authentication logic

	// Send the authentication response
	var response models.AuthResponsePayload
	if authenticated {
		response = models.AuthResponsePayload{
			Status: variables.AuthSuccess,
		}

		// Store the client's connection information after successful authentication
		client := &Client{
			Username: payload.Username,
			Conn:     conn,
			State:    AUTHENTICATED,
		}
		s.AddClient(client)

		// Send the authentication response with header
		header := models.Header{
			Version:  variables.Version,
			Type:     variables.AuthResponse,
			Length:   uint16(binary.Size(response)),
			Sequence: 0, // Sequence number, update this as needed
		}

		// Write header
		err := binary.Write(conn, binary.BigEndian, &header)
		if err != nil {
			log.Printf("Failed to send header to client: %v", err)
			return
		}

		// Write payload
		err = binary.Write(conn, binary.BigEndian, &response)
		if err != nil {
			log.Printf("Failed to send AUTH_RESPONSE to client: %v", err)
			return
		}
	} else {
		response = models.AuthResponsePayload{
			Status: variables.AuthFailure,
		}

		// Send the authentication response with header
		header := models.Header{
			Version:  variables.Version,
			Type:     variables.AuthResponse,
			Length:   uint16(binary.Size(response)),
			Sequence: 0, // Sequence number, update this as needed
		}

		// Write header
		err := binary.Write(conn, binary.BigEndian, &header)
		if err != nil {
			log.Printf("Failed to send header to client: %v", err)
			return
		}

		// Write payload
		err = binary.Write(conn, binary.BigEndian, &response)
		if err != nil {
			log.Printf("Failed to send AUTH_RESPONSE to client: %v", err)
			return
		}
	}
}

func (s *Server) AddClient(client *Client) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.Clients[string(bytes.Trim(client.Username[:], "\x00"))] = client
}

func (s *Server) StoreCertificate(client *Client, cert [512]byte) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.Clients[string(bytes.Trim(client.Username[:], "\x00"))].Key = cert
}

func (s *Server) HandleKeyExchange(conn net.Conn, payload models.PublicKeyPayload) error {
	username := string(bytes.Trim(payload.Username[:], "\x00"))
	// Store the client's public key

	client, ok := s.Clients[username]

	if !ok {
		return fmt.Errorf("unknown client: %s", username)
	}

	if client.State < AUTHENTICATED {
		return fmt.Errorf("user not Authenticated: %s", username)
	}

	s.StoreCertificate(client, payload.Key)

	s.mutex.Lock()
	client.State = PUBLIC_KEY_RECVD
	s.mutex.Unlock()

	// Send the public keys of other clients to the newly connected client
	header := models.Header{
		Version:  variables.Version,
		Type:     variables.KeyExchange,
		Sequence: 0, // Sequence number, update this as needed
	}

	// Send public keys of existing clients to the new client
	for _, otherClient := range s.Clients {
		if otherClient.Username != payload.Username {
			publicKeyPayload := models.PublicKeyPayload{
				Username: otherClient.Username,
				Key:      otherClient.Key,
			}

			// Write header
			err := binary.Write(conn, binary.BigEndian, &header)
			if err != nil {
				return fmt.Errorf("failed to send header to client: %v", err)
			}

			// Write payload
			err = binary.Write(conn, binary.BigEndian, &publicKeyPayload)
			if err != nil {
				return fmt.Errorf("failed to send PUBLIC_KEY to client: %v", err)
			}
		}
	}

	// Send the new client's public key to existing clients
	for _, otherClient := range s.Clients {
		if otherClient.Username != payload.Username {
			publicKeyPayload := models.PublicKeyPayload{
				Username: client.Username,
				Key:      client.Key,
			}

			// Write header
			err := binary.Write(otherClient.Conn, binary.BigEndian, &header)
			if err != nil {
				return fmt.Errorf("failed to send header to client: %v", err)
			}

			// Write payload
			err = binary.Write(otherClient.Conn, binary.BigEndian, &publicKeyPayload)
			if err != nil {
				return fmt.Errorf("failed to send PUBLIC_KEY to client: %v", err)
			}
		}
	}
	s.mutex.Lock()
	client.State = PUBLIC_KEY_SENT
	s.mutex.Unlock()
	return nil
}

func (s *Server) HandleMessage(conn net.Conn, payload models.MessagePayload) error {
	// Extract sender, recipient, and message text from the payload
	sender := string(bytes.Trim(payload.Sender[:], "\x00"))
	recipient := string(bytes.Trim(payload.Recipient[:], "\x00"))
	messageText := string(bytes.Trim(payload.Data[:], "\x00"))

	// Look up the sender and recipient's connection
	s.mutex.Lock()
	senderClient, ok := s.Clients[sender]

	if !ok {
		return fmt.Errorf("unknown sender: %s", sender)
	}

	if senderClient.State != PUBLIC_KEY_SENT && senderClient.State != CHAT {
		return fmt.Errorf("unknown sender: %s", sender)
	}

	recipientClient, ok := s.Clients[recipient]

	if !ok {
		return fmt.Errorf("unknown recipient: %s", recipient)
	}

	if recipientClient.State != PUBLIC_KEY_SENT && recipientClient.State != CHAT {
		return fmt.Errorf("unknown recipient: %s", recipient)
	}

	senderClient.State = CHAT
	recipientClient.State = CHAT
	s.mutex.Unlock()

	// Send the message to the recipient
	header := models.Header{
		Version:  variables.Version,
		Type:     variables.Message,
		Length:   uint16(binary.Size(payload)),
		Sequence: 0, // Sequence number, update this as needed
	}

	// Write header
	err := binary.Write(recipientClient.Conn, binary.BigEndian, &header)
	if err != nil {
		return fmt.Errorf("failed to send header to recipient: %v", err)
	}

	// Write payload
	err = binary.Write(recipientClient.Conn, binary.BigEndian, &payload)
	if err != nil {
		return fmt.Errorf("failed to send MESSAGE to recipient: %v", err)
	}

	// Print the received message
	log.Printf("Message from %s to %s: %s\n", sender, recipient, messageText)

	return nil
}

func (s *Server) HandleDisconnect(conn net.Conn, payload models.DisconnectPayload) error {
	// Close the connection
	conn.Close()

	// Find and remove the client from the clients map
	var disconnectedClient *Client
	var disconnectedUsername string

	s.mutex.Lock()

	for username, client := range s.Clients {
		if client.Conn == conn {
			disconnectedClient = client
			disconnectedClient.State = DISCONNECTING
			disconnectedUsername = username
			delete(s.Clients, username)
			disconnectedClient.State = TERMINATED
			break
		}
	}

	s.mutex.Unlock()

	header := models.Header{
		Version:  variables.Version,
		Type:     variables.KeyExchange,
		Sequence: 0, // Sequence number, update this as needed
	}

	// Inform other clients about disconnect.
	for _, otherClient := range s.Clients {
		if otherClient.Username != disconnectedClient.Username {
			publicKeyPayload := models.PublicKeyPayload{
				Username: disconnectedClient.Username,
			}

			// Write header
			err := binary.Write(otherClient.Conn, binary.BigEndian, &header)
			if err != nil {
				return fmt.Errorf("failed to send header to client: %v", err)
			}

			// Write payload
			err = binary.Write(otherClient.Conn, binary.BigEndian, &publicKeyPayload)
			if err != nil {
				return fmt.Errorf("failed to send PUBLIC_KEY to client: %v", err)
			}
		}
	}

	// Print the disconnection message
	if disconnectedClient != nil && disconnectedClient.State == TERMINATED {
		fmt.Printf("Client %s disconnected\n", disconnectedUsername)
		disconnectedClient.Conn.Close()
	}
	return nil
}
