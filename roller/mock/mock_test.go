package mock

//var (
//	cfg        *config.Config
//	scrollPort = 9020
//	mockPath   string
//)

// func TestMain(m *testing.M) {
// 	mockPath = "/tmp/roller_mock_test"
// 	_ = os.RemoveAll(mockPath)
// 	if err := os.Mkdir(mockPath, os.ModePerm); err != nil {
// 		fmt.Fprintln(os.Stderr, err)
// 		os.Exit(1)
// 	}
// 	scrollPort = rand.Intn(9000)
// 	cfg = &config.Config{
// 		RollerName: "test-roller",
// 		SecretKey:  "dcf2cbdd171a21c480aa7f53d77f31bb102282b3ff099c78e3118b37348c72f7",
// 		ScrollURL:  fmt.Sprintf("ws://localhost:%d", scrollPort),
// 		Prover:     &config.ProverConfig{MockMode: true},
// 		DBPath:     filepath.Join(mockPath, "stack_db"),
// 	}

// 	os.Exit(m.Run())
// }
// func TestRoller(t *testing.T) {
// 	go mockScroll(t)

// 	r, err := roller.NewRoller(cfg)
// 	assert.NoError(t, err)
// 	go r.Run()

// 	<-time.NewTimer(2 * time.Second).C
// 	r.Close()
// }

// func mockScroll(t *testing.T) {
// 	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
// 		up := websocket.Upgrader{}
// 		c, err := up.Upgrade(w, req, nil)
// 		assert.NoError(t, err)

// 		var payload []byte
// 		payload, err = func(c *websocket.Conn) ([]byte, error) {
// 			for {
// 				var mt int
// 				mt, payload, err = c.ReadMessage()
// 				if err != nil {
// 					return nil, err
// 				}

// 				if mt == websocket.BinaryMessage {
// 					return payload, nil
// 				}
// 			}
// 		}(c)
// 		assert.NoError(t, err)

// 		msg := &Msg{}
// 		err = json.Unmarshal(payload, msg)
// 		assert.NoError(t, err)

// 		authMsg := &AuthMessage{}
// 		err = json.Unmarshal(msg.Payload, authMsg)
// 		assert.NoError(t, err)

// 		// Verify signature
// 		hash, err := authMsg.Identity.Hash()
// 		assert.NoError(t, err)

// 		if !secp256k1.VerifySignature(common.FromHex(authMsg.Identity.PublicKey), hash, common.FromHex(authMsg.Signature)[:64]) {
// 			assert.NoError(t, err)
// 		}
// 		t.Log("signature verification successful. Roller: ", authMsg.Identity.Name)
// 		assert.Equal(t, cfg.RollerName, authMsg.Identity.Name)

// 		traces := &BlockTraces{
// 			ID:     16,
// 			Traces: nil,
// 		}
// 		msgByt, err := roller.MakeMsgByt(BlockTrace, traces)
// 		assert.NoError(t, err)

// 		err = c.WriteMessage(websocket.BinaryMessage, msgByt)
// 		assert.NoError(t, err)
// 	})
// 	http.ListenAndServe(fmt.Sprintf(":%d", scrollPort), nil)
// }
