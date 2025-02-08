// Set the base URL for the API request
const baseUrl = `${window.location.origin}${window.location.pathname}`;

// Function to handle the page load event
window.onload = () => {
    const qrCodeEl = document.getElementById('qrcode');
    const linkButton = document.getElementById('button');
    let sessionId = '1'; // Match the sessionId from GetAuthRequest

    // Start polling immediately when page loads
    startStatusPolling(sessionId);

    fetch(`${baseUrl}api/sign-in`)
        .then(response => {
            if (response.ok) {
                return response.json();
            } else {
                throw new Error('Failed to fetch API data');
            }
        })
        .then(data => {
            // Generate QR code
            generateQrCode(qrCodeEl, data);
            qrCodeEl.style.display = 'block'; // Show the QR code

            // Encode the data in Base64 for the universal link
            const encodedRequest = btoa(JSON.stringify(data));
            linkButton.href = `https://wallet.privado.id/#i_m=${encodedRequest}`;
            linkButton.style.display = 'block'; // Show the universal link button
        })
        .catch(error => console.error('Error fetching data from API:', error));
};

// Helper function to generate QR code
function generateQrCode(element, data) {
    new QRCode(element, {
        text: JSON.stringify(data),
        width: 256,
        height: 256,
        correctLevel: QRCode.CorrectLevel.Q // Error correction level
    });
}

// Function to poll verification status
function startStatusPolling(sessionId) {
    const pollInterval = 3000; // 3 seconds
    console.log('Starting status polling for sessionId:', sessionId);
    
    const statusCheck = setInterval(() => {
        console.log('Polling status for sessionId:', sessionId);
        fetch(`${baseUrl}api/status?sessionId=${sessionId}`)
            .then(response => {
                console.log('Response status:', response.status);
                return response.json();
            })
            .then(data => {
                console.log('Polling response data:', data);
                if (data.status === 'success') {
                    console.log('Verification completed:', data);
                    clearInterval(statusCheck); // Stop polling once verified
                    // You can add UI updates here based on verification success
                    checkVerificationStatus();
                } else {
                    console.log('Still waiting for verification. Current status:', data.status);
                }
            })
            .catch(error => console.error('Error checking status:', error));
    }, pollInterval);

    return statusCheck; // Return the interval ID
}

function checkVerificationStatus() {
    fetch(`${baseUrl}api/status?sessionId=1`)
        .then(response => response.json())
        .then(data => {
            if (data.status === "success") {
                // Hide QR code and show verification result
                document.querySelector('.qr-container').style.display = 'none';
                const verificationResult = document.getElementById('verification-result');
                verificationResult.style.display = 'block';

                // Update status
                const resultStatus = verificationResult.querySelector('.result-status');
                resultStatus.textContent = "✅ Verification Successful!";
                resultStatus.classList.add('success');

                // Display JWT
                if (data.data) {
                    console.log('JWT data:', data.data);
                    const jwtDisplay = document.getElementById('jwt-display');
                    jwtDisplay.textContent = data.data;
                }

                // Log the data
                console.log('Verification data:', data);
            }
        })
        .catch(error => console.error('Error:', error));
}

// Function to start a new verification process
function startVerification() {
    // Reset verification results
    const sessionId = '1';
    
    // Start polling
    startStatusPolling(sessionId);

    fetch(`${baseUrl}api/sign-in`)
        .then(response => {
            if (response.ok) {
                return response.json();
            } else {
                throw new Error('Failed to fetch API data');
            }
        })
        .then(data => {
            // Generate QR code
            const qrCodeEl = document.getElementById('qrcode');
            qrCodeEl.innerHTML = ''; // Clear existing QR code
            generateQrCode(qrCodeEl, data);
            qrCodeEl.style.display = 'block';

            // Update universal link
            const linkButton = document.getElementById('button');
            const encodedRequest = btoa(JSON.stringify(data));
            linkButton.href = `https://wallet.privado.id/#i_m=${encodedRequest}`;
            linkButton.style.display = 'block';

            // Reset JWT display
            const jwtDisplay = document.getElementById('jwt-display');
            jwtDisplay.textContent = '';
        })
        .catch(error => console.error('Error fetching data from API:', error));
}

// Add event listener for "Verify Again" button
document.getElementById('verify-again').addEventListener('click', () => {
    // Reset the UI
    document.querySelector('.qr-container').style.display = 'block';
    document.getElementById('verification-result').style.display = 'none';
    
    // Restart the verification process
    startVerification();
});