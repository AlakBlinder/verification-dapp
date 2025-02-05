package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	circuits "github.com/iden3/go-circuits/v2"
	auth "github.com/iden3/go-iden3-auth/v2"

	"github.com/iden3/go-iden3-auth/v2/pubsignals"
	"github.com/iden3/go-iden3-auth/v2/state"
	"github.com/iden3/iden3comm/v2/protocol"
)

const VerificationKeyPath = "verification_key.json"

type KeyLoader struct {
	Dir string
}

// Load keys from embedded FS
func (m KeyLoader) Load(id circuits.CircuitID) ([]byte, error) {
	return os.ReadFile(fmt.Sprintf("%s/%v/%s", m.Dir, id, VerificationKeyPath))
}

func main() {
	fs := http.FileServer(http.Dir("../static"))
	http.Handle("/", fs)
	http.HandleFunc("/api/sign-in", GetAuthRequest)
	http.HandleFunc("/api/callback", Callback)
	log.Println("Starting server at port 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

// Create a map to store the auth requests and their session IDs
var requestMap = make(map[string]interface{})

func GetAuthRequest(w http.ResponseWriter, r *http.Request) {

	// Audience is verifier id
	rURL := "https://f99a-2601-642-4f7c-f40-78dc-2db2-831d-9422.ngrok-free.app"
	sessionID := 1
	CallbackURL := "/api/callback"
	Audience := "did:iden3:polygon:amoy:x6x5sor7zpxhPBRFEZXv8dKoxpEibsDHHhFAaCbne"

	uri := fmt.Sprintf("%s%s?sessionId=%s", rURL, CallbackURL, strconv.Itoa(sessionID))

	// Generate request for basic authentication
	var request protocol.AuthorizationRequestMessage = auth.CreateAuthorizationRequest("test flow", Audience, uri)

	// Add request for a specific proof
	var mtpProofRequest protocol.ZeroKnowledgeProofRequest
	mtpProofRequest.ID = 1
	mtpProofRequest.CircuitID = string(circuits.AtomicQueryMTPV2CircuitID)
	mtpProofRequest.Query = map[string]interface{}{
		"allowedIssuers": []string{"*"},
		"credentialSubject": map[string]interface{}{
			"walletAddress": map[string]interface{}{},
		},
		"context": "ipfs://QmdGrFoZrEgUoiS4QN77YSWY5LfcQDKQAzBTtkG5dLw1YV",
		"type":    "SocialCredential",
	}
	request.Body.Scope = append(request.Body.Scope, mtpProofRequest)

	// Store auth request in map associated with session ID
	requestMap[strconv.Itoa(sessionID)] = request

	// print request
	fmt.Println(request)

	msgBytes, _ := json.Marshal(request)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(msgBytes)
	return
}

// Callback works with sign-in callbacks
func Callback(w http.ResponseWriter, r *http.Request) {
	fmt.Println("callback")
	// Get session ID from request
	sessionID := r.URL.Query().Get("sessionId")

	// get JWZ token params from the post request
	tokenBytes, err := io.ReadAll(r.Body)

	if err != nil {
		log.Println(err)
		return
	}

	// Add Polygon Mumbai RPC node endpoint - needed to read on-chain state
	ethURL := "https://rpc.ankr.com/polygon_amoy/6f897086c192bc30e5f61db622983e55c342ef4de3cd0eb9c4f5eaecb9f623d6"

	// Add identity state contract address
	contractAddress := "0x1a4cC30f2aA0377b0c3bc9848766D90cb4404124"

	resolverPrefix := "polygon:amoy"

	// Locate the directory that contains circuit's verification keys
	keyDIR := "../keys"

	// fetch authRequest from sessionID
	authRequest := requestMap[sessionID]

	// print authRequest
	log.Println(authRequest)

	// load the verifcation key
	var verificationKeyLoader = &KeyLoader{Dir: keyDIR}

	resolver := map[string]state.ETHResolver{
		"polygon:amoy": {
			RPCUrl:          ethURL,
			ContractAddress: common.HexToAddress(contractAddress),
		},
		"privado:main": {
			RPCUrl:          "https://rpc-mainnet.privado.id",
			ContractAddress: common.HexToAddress("0x3C9acB2205Aa72A05F6D77d708b5Cf85FCa3a896"),
		},
	}

	resolvers := map[string]pubsignals.StateResolver{
		resolverPrefix: resolver["polygon:amoy"],
	}

	// EXECUTE VERIFICATION
	verifier, err := auth.NewVerifier(verificationKeyLoader, resolvers, auth.WithIPFSGateway("https://ipfs.io"))
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Println("verifier created")
	authResponse, err := verifier.FullVerify(
		r.Context(),
		string(tokenBytes),
		authRequest.(protocol.AuthorizationRequestMessage),
		pubsignals.WithAcceptedStateTransitionDelay(time.Minute*5))
	if err != nil {
		log.Println("error verifying auth response")
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Println("auth response verified")

	// Create a response structure
	response := struct {
		Status     string                 `json:"status"`
		Message    string                 `json:"message"`
		Verified   bool                   `json:"verified"`
		Attributes map[string]interface{} `json:"attributes,omitempty"`
	}{
		Status:   "success",
		Message:  "Verification passed successfully",
		Verified: true,
		Attributes: map[string]interface{}{
			"proof": authResponse.Body.Scope[0].Proof,
		},
	}

	messageBytes, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Failed to create response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(messageBytes)
	log.Println("verification passed")
}
