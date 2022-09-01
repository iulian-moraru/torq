package torqsrv

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/jmoiron/sqlx"
	"github.com/lncapital/torq/internal/channels"
	"github.com/lncapital/torq/internal/payments"
	"github.com/lncapital/torq/pkg/server_errors"
)

type wsRequest struct {
	ReqId               string                        `json:"reqId"`
	Type                string                        `json:"type"`
	NewPaymentRequest   *payments.NewPaymentRequest   `json:"newPaymentRequest"`
	OpenChannelRequest  *channels.OpenChannelRequest  `json:"openChannelRequest"`
	CloseChannelRequest *channels.CloseChannelRequest `json:"closeChannelRequest"`
	Password            *string                       `json:"password"`
}

type Pong struct {
	Message string `json:"message"`
}

type AuthSuccess struct {
	AuthSuccess bool `json:"authSuccess"`
}

type wsError struct {
	ReqId string `json:"id"`
	Type  string `json:"type"`
	Error string `json:"error"`
}

func processWsReq(db *sqlx.DB, c *gin.Context, wChan chan interface{}, req wsRequest) {
	if req.Type == "ping" {
		wChan <- Pong{Message: "pong"}
		return
	}

	// Validate request
	if req.ReqId == "" {
		wChan <- wsError{
			ReqId: req.ReqId,
			Type:  "Error",
			Error: "ReqId cannot be empty",
		}
		return
	}

	switch req.Type {
	case "auth":
		wChan <- wsError{
			ReqId: req.ReqId,
			Type:  "Error",
			Error: "You are already authenticated",
		}
	case "newPayment":
		if req.NewPaymentRequest == nil {
			wChan <- wsError{
				ReqId: req.ReqId,
				Type:  "Error",
				Error: "newPaymentRequest cannot be empty",
			}
			break
		}
		// Process a valid payment request
		err := payments.SendNewPayment(wChan, db, c, *req.NewPaymentRequest, req.ReqId)
		if err != nil {
			wChan <- wsError{
				ReqId: req.ReqId,
				Type:  "Error",
				Error: err.Error(),
			}
		}
		break
	case "closeChannel":
		if req.CloseChannelRequest == nil {
			wChan <- wsError{
				ReqId: req.ReqId,
				Type:  "Error",
				Error: "Close Channel request cannot be empty",
			}
			break
		}
		// Process a valid payment request
		err := channels.CloseChannel(wChan, db, c, *req.CloseChannelRequest, req.ReqId)
		if err != nil {
			wChan <- wsError{
				ReqId: req.ReqId,
				Type:  "Error",
				Error: err.Error(),
			}
		}
	case "openChannel":
		if req.OpenChannelRequest == nil {
			wChan <- wsError{
				ReqId: req.ReqId,
				Type:  "Error",
				Error: "OpenChannelRequest cannot be empty",
			}
			break
		}
		err := channels.OpenChannel(db, wChan, *req.OpenChannelRequest, req.ReqId)
		if err != nil {
			wChan <- wsError{
				ReqId: req.ReqId,
				Type:  "Error",
				Error: err.Error(),
			}
		}
		break
	default:
		err := fmt.Errorf("Unknown request type: %s", req.Type)
		wChan <- wsError{
			ReqId: req.ReqId,
			Type:  "Error",
			Error: err.Error(),
		}
	}
}

func WebsocketHandler(c *gin.Context, db *sqlx.DB, apiPwd string) {

	conn, err := wsUpgrade.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		server_errors.LogAndSendServerError(c, err)
		return
	}
	defer conn.Close()

	// Channel for writing responses to the client safely
	wc := make(chan interface{})
	go func(c *gin.Context) {
		for {
			err := conn.WriteJSON(<-wc)
			if err != nil {
				server_errors.LogAndSendServerError(c, err)
			}
		}
	}(c)

	// Boolean indicating whether the client is authenticated
	allowedUser := false

	for {
		req := wsRequest{}
		err := conn.ReadJSON(&req)
		switch err.(type) {
		case *websocket.CloseError:
			return
		case *websocket.HandshakeError:
			server_errors.LogAndSendServerError(c, err)
			return
		case nil:
			// Check if the client is authenticated
			if allowedUser == false {
				if req.Type != "auth" {
					wc <- wsError{
						ReqId: req.ReqId,
						Type:  "Error",
						Error: "Unauthorized. Please login first.",
					}
					continue
				}
				if *req.Password == apiPwd {
					allowedUser = true
					wc <- AuthSuccess{AuthSuccess: true}
					continue
				}
				wc <- wsError{
					ReqId: req.ReqId,
					Type:  "Error",
					Error: "Incorrect password",
				}
				continue
			}

			go processWsReq(db, c, wc, req)
			continue
		default:
			wsr := wsError{
				ReqId: req.ReqId,
				Type:  "Error",
				Error: fmt.Sprintf("Could not parse request, please check that your JSON is correctly formated"),
			}
			wc <- wsr
			continue
		}

	}
}
