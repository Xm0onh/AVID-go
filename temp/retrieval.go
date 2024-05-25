package main

// func RetrieveData(ctx context.Context, h host.Host, n, k int, method string) ([]byte, error) {
// 	chunks := make([][]byte, 0, n)
// 	hashes := make(map[[sha256.Size]byte]struct{})

// 	for _, peerID := range h.Network().Peers() {
// 		fmt.Printf("Sending handshake to peer %s\n", peerID.String())
// 		s, err := h.NewStream(ctx, peerID, protocolID)
// 		if err != nil {
// 			fmt.Println("Error creating stream:", err)
// 			continue
// 		}

// 		_, err = s.Write([]byte("handshake "))
// 		if err != nil {
// 			fmt.Println("Error sending handshake request:", err)
// 			s.Close()
// 			continue
// 		}

// 		buf := new(bytes.Buffer)
// 		io.Copy(buf, s)
// 		data := buf.Bytes()

// 		if string(data) != "handshake ack" {
// 			fmt.Println("Handshake failed with peer", peerID)
// 			s.Close()
// 			continue
// 		}
// 		s.Close()
// 	}

// 	for _, peerID := range h.Network().Peers() {
// 		fmt.Printf("Requesting data from peer %s\n", peerID.String())
// 		s, err := h.NewStream(ctx, peerID, protocolID)
// 		if err != nil {
// 			fmt.Println("Error creating stream:", err)
// 			continue
// 		}

// 		_, err = s.Write([]byte("retrieve "))
// 		if err != nil {
// 			fmt.Println("Error sending retrieve request:", err)
// 			s.Close()
// 			continue
// 		}

// 		buf := new(bytes.Buffer)
// 		io.Copy(buf, s)
// 		data := buf.Bytes()

// 		fmt.Printf("Received data of length %d from peer %s\n", len(data), peerID.String())

// 		for len(data) > 0 {
// 			if len(data) < sha256.Size {
// 				fmt.Println("Received data is too short from peer", peerID)
// 				break
// 			}

// 			hash := [sha256.Size]byte{}
// 			copy(hash[:], data[:sha256.Size])
// 			data = data[sha256.Size:]

// 			var chunk []byte
// 			if len(data) >= 1024-sha256.Size {
// 				chunk = data[:1024-sha256.Size]
// 				data = data[1024-sha256.Size:]
// 			} else {
// 				chunk = data
// 				data = nil
// 			}

// 			if sha256.Sum256(chunk) != hash {
// 				fmt.Println("Hash mismatch for chunk from", peerID)
// 				continue
// 			}

// 			if _, exists := hashes[hash]; !exists {
// 				hashes[hash] = struct{}{}
// 				chunks = append(chunks, chunk)
// 			}

// 			if len(chunks) >= k {
// 				break
// 			}
// 		}

// 		if len(chunks) >= k {
// 			break
// 		}
// 	}

// 	if len(chunks) < k {
// 		return nil, errors.New("not enough chunks retrieved")
// 	}

// 	if method == "RS" {
// 		return DecodeRS(chunks, k)
// 	} else {
// 		return nil, fmt.Errorf("unknown method: %s", method)
// 	}
// }
