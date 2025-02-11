package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	circuits "github.com/iden3/go-circuits/v2"
	auth "github.com/iden3/go-iden3-auth/v2"

	"github.com/iden3/go-iden3-auth/v2/pubsignals"
	"github.com/iden3/go-iden3-auth/v2/state"
	"github.com/iden3/iden3comm/v2/protocol"
)

// Configuration constants
const (
	// Base URL for the application
	BaseURL = "https://verification-dapp-v1-1049789873803.us-west1.run.app"

	// Callback endpoint
	CallbackURL = "/api/callback"

	// Verifier ID
	Audience = "did:iden3:polygon:amoy:x6x5sor7zpxhPBRFEZXv8dKoxpEibsDHHhFAaCbne"

	// Verification key path
	VerificationKeyPath = "verification_key.json"

	// Polygon RPC endpoint
	EthURL = "https://rpc.ankr.com/polygon_amoy/6f897086c192bc30e5f61db622983e55c342ef4de3cd0eb9c4f5eaecb9f623d6"

	// Contract address for identity state
	ContractAddress = "0x1a4cC30f2aA0377b0c3bc9848766D90cb4404124"

	// Resolver prefix for Polygon network
	ResolverPrefix = "polygon:amoy"

	// Directory containing circuit verification keys
	KeyDir = "./keys"

	// IPFS gateway
	IpfsGateway = "https://ipfs.io"
)

type KeyLoader struct {
	Dir string
}

// Load keys from embedded FS
func (m KeyLoader) Load(id circuits.CircuitID) ([]byte, error) {
	return os.ReadFile(fmt.Sprintf("%s/%v/%s", m.Dir, id, VerificationKeyPath))
}

func main() {
	// Get PORT from environment (default to 8080 if not set)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)
	http.HandleFunc("/api/sign-in", GetAuthRequest)
	http.HandleFunc("/api/callback", Callback)
	http.HandleFunc("/api/status", GetVerificationStatus)
	log.Println("Starting server at port 8080")
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

// Create a map to store the auth requests and their session IDs
var requestMap = make(map[string]interface{})

func GetAuthRequest(w http.ResponseWriter, r *http.Request) {
	// Clear any existing verification result for this session
	sessionID := "1" // Since we're using a fixed session ID
	delete(verificationResults, sessionID)

	uri := fmt.Sprintf("%s%s?sessionId=%s", BaseURL, CallbackURL, sessionID)

	// Generate request for basic authentication
	var request protocol.AuthorizationRequestMessage = auth.CreateAuthorizationRequest("Verify your Social Credential", Audience, uri)

	// Add request for a specific proof
	var walletProofRequest protocol.ZeroKnowledgeProofRequest
	walletProofRequest.ID = 1
	walletProofRequest.CircuitID = string(circuits.AtomicQueryMTPV2CircuitID)
	walletProofRequest.Query = map[string]interface{}{
		"allowedIssuers": []string{"*"},
		"credentialSubject": map[string]interface{}{
			"walletAddress": map[string]interface{}{},
		},
		"context": "ipfs://QmdGrFoZrEgUoiS4QN77YSWY5LfcQDKQAzBTtkG5dLw1YV",
		"type":    "SocialCredential",
	}
	request.Body.Scope = append(request.Body.Scope, walletProofRequest)

	var emailProofRequest protocol.ZeroKnowledgeProofRequest
	emailProofRequest.ID = 2
	emailProofRequest.CircuitID = string(circuits.AtomicQueryMTPV2CircuitID)
	emailProofRequest.Query = map[string]interface{}{
		"allowedIssuers": []string{"*"},
		"credentialSubject": map[string]interface{}{
			"email": map[string]interface{}{},
		},
		"context": "ipfs://QmdGrFoZrEgUoiS4QN77YSWY5LfcQDKQAzBTtkG5dLw1YV",
		"type":    "SocialCredential",
	}
	request.Body.Scope = append(request.Body.Scope, emailProofRequest)

	// Store auth request in map associated with session ID
	requestMap[sessionID] = request

	// print request
	fmt.Println(request)

	msgBytes, _ := json.Marshal(request)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(msgBytes)
	return
}

// Store verification results in memory (or database)
var verificationResults = make(map[string]interface{})

// Callback works with sign-in callbacks
func Callback(w http.ResponseWriter, r *http.Request) {
	fmt.Println("callback")
	// Get session ID from request
	sessionID := r.URL.Query().Get("sessionId")

	// get JWZ token params from the post request
	tokenBytes, err := io.ReadAll(r.Body)
	// Convert tokenBytes to string to see the JWT
	fmt.Println("JWT Token:", string(tokenBytes))

	// Store result
	verificationResults[sessionID] = tokenBytes

	if err != nil {
		log.Println(err)
		return
	}

	// Add Polygon Mumbai RPC node endpoint - needed to read on-chain state
	ethURL := EthURL

	// Add identity state contract address
	contractAddress := ContractAddress

	resolverPrefix := ResolverPrefix

	// Locate the directory that contains circuit's verification keys
	keyDIR := KeyDir

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
	}

	resolvers := map[string]pubsignals.StateResolver{
		resolverPrefix: resolver["polygon:amoy"],
	}

	// EXECUTE VERIFICATION
	verifier, err := auth.NewVerifier(verificationKeyLoader, resolvers, auth.WithIPFSGateway(IpfsGateway))
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Println("verifier created")
	_, err = verifier.FullVerify(
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
		Status   string `json:"status"`
		Message  string `json:"message"`
		Verified bool   `json:"verified"`
		JWT      string `json:"jwt"`
	}{
		Status:   "success",
		Message:  "Verification passed successfully",
		Verified: true,
		JWT:      string(tokenBytes),
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

// Add new endpoint for frontend to check status
func GetVerificationStatus(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("sessionId")

	w.Header().Set("Content-Type", "application/json")

	result, exists := verificationResults[sessionID]
	if !exists {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "pending",
			"message": "Verification in progress...",
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Verification completed",
		"data":    result,
	})
}
