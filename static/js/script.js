document.addEventListener('DOMContentLoaded', function() {
    // Handle password visibility toggle for all password fields
    window.setupPasswordToggle = function(passwordId, toggleId) {
        const passwordField = document.getElementById(passwordId);
        const toggleButton = document.getElementById(toggleId);
        
        if (toggleButton && passwordField) {
            toggleButton.addEventListener('click', function() {
                const type = passwordField.getAttribute('type') === 'password' ? 'text' : 'password';
                passwordField.setAttribute('type', type);
                
                // Update icon
                if (type === 'password') {
                    toggleButton.innerHTML = '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path><circle cx="12" cy="12" r="3"></circle></svg>';
                } else {
                    toggleButton.innerHTML = '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"></path><line x1="1" y1="1" x2="23" y2="23"></line></svg>';
                }
            });
        }
    };
    
    // Setup password toggle for login form
    setupPasswordToggle('password', 'togglePassword');
    
    // Setup password toggle for register form
    setupPasswordToggle('registerPassword', 'toggleRegisterPassword');
    setupPasswordToggle('confirmPassword', 'toggleConfirmPassword');
    
    // Form validation
    const loginForm = document.getElementById('loginForm');
    if (loginForm) {
        loginForm.addEventListener('submit', function(event) {
            const username = document.getElementById('username').value.trim();
            const password = document.getElementById('password').value.trim();
            let valid = true;
            
            // Reset error messages
            document.querySelectorAll('.error-message').forEach(function(el) {
                el.remove();
            });
            
            if (!username) {
                valid = false;
                showError('username', 'Username is required');
            }
            
            if (!password) {
                valid = false;
                showError('password', 'Password is required');
            }
            
            if (!valid) {
                event.preventDefault();
            }
        });
    }
    
    // Register form validation
    const registerForm = document.getElementById('registerForm');
    if (registerForm) {
        registerForm.addEventListener('submit', function(event) {
            // Enable validation instead of disabling it
            let valid = true;
            
            // Reset error messages
            document.querySelectorAll('.error-message').forEach(function(el) {
                el.remove();
            });
            
            // Get form values
            const username = document.getElementById('registerUsername').value.trim();
            const email = document.getElementById('email').value.trim();
            const password = document.getElementById('registerPassword').value;
            const confirmPassword = document.getElementById('confirmPassword').value;
            const dob = document.getElementById('dob').value.trim();
            const ssn = document.getElementById('ssn').value.trim();
            
            // Validate username
            if (!username) {
                valid = false;
                showError('registerUsername', 'Username is required');
            }
            
            // Validate email
            if (!email) {
                valid = false;
                showError('email', 'Email is required');
            } else if (!isValidEmail(email)) {
                valid = false;
                showError('email', 'Please enter a valid email address');
            }
            
            // Validate SSN
            if (!ssn) {
                valid = false;
                showError('ssn', 'SSN is required');
            } else if (!isValidSSN(ssn)) {
                valid = false;
                showError('ssn', 'Please enter a valid SSN');
            }
            
            // Validate DOB
            if (!dob) {
                valid = false;
                showError('dob', 'Date of Birth is required');
            }
            
            // Validate password
            if (!password) {
                valid = false;
                showError('registerPassword', 'Password is required');
            } else if (password.length < 8) {
                valid = false;
                showError('registerPassword', 'Password must be at least 8 characters');
            }
            
            // Validate password confirmation
            if (!confirmPassword) {
                valid = false;
                showError('confirmPassword', 'Please confirm your password');
            } else if (password !== confirmPassword) {
                valid = false;
                showError('confirmPassword', 'Passwords do not match');
            }
            
            if (!valid) {
                event.preventDefault();
            }
        });
    }
    
    // Forgot password form validation
    const forgotForm = document.getElementById('forgotForm');
    if (forgotForm) {
        forgotForm.addEventListener('submit', function(event) {
            const email = document.getElementById('email').value.trim();
            let valid = true;
            
            // Reset error messages
            document.querySelectorAll('.error-message').forEach(function(el) {
                el.remove();
            });
            
            if (!email) {
                valid = false;
                showError('email', 'Email is required');
            } else if (!isValidEmail(email)) {
                valid = false;
                showError('email', 'Please enter a valid email address');
            }
            
            if (!valid) {
                event.preventDefault();
            }
        });
    }
    
    // Helper to show error messages
    function showError(fieldId, message) {
        const field = document.getElementById(fieldId);
        const errorDiv = document.createElement('div');
        errorDiv.className = 'error-message';
        errorDiv.style.color = 'var(--error-color)';
        errorDiv.style.fontSize = '0.75rem';
        errorDiv.style.marginTop = '0.25rem';
        errorDiv.textContent = message;
        field.parentNode.appendChild(errorDiv);
    }
    
    // Email validation helper
    function isValidEmail(email) {
        const re = /^(([^<>()\[\]\\.,;:\s@"]+(\.[^<>()\[\]\\.,;:\s@"]+)*)|(".+"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$/;
        return re.test(String(email).toLowerCase());
    }
    
    // SSN validation helper
    function isValidSSN(ssn) {
        // Remove any non-digit characters
        const digitsOnly = ssn.replace(/\D/g, '');
        
        // Check if it's 9 digits
        if (digitsOnly.length !== 9) {
            return false;
        }
        
        // Check for invalid SSN patterns
        // First 3 digits can't be 000, 666, or 900-999
        const firstThree = parseInt(digitsOnly.substring(0, 3));
        if (firstThree === 0 || firstThree === 666 || (firstThree >= 900 && firstThree <= 999)) {
            return false;
        }
        
        // Middle 2 digits can't be 00
        const middleTwo = parseInt(digitsOnly.substring(3, 5));
        if (middleTwo === 0) {
            return false;
        }
        
        // Last 4 digits can't be 0000
        const lastFour = parseInt(digitsOnly.substring(5, 9));
        if (lastFour === 0) {
            return false;
        }
        
        return true;
    }
    
    // Auto-hide alerts after 5 seconds
    const alerts = document.querySelectorAll('.alert');
    alerts.forEach(function(alert) {
        setTimeout(function() {
            alert.style.opacity = '0';
            alert.style.transition = 'opacity 0.5s ease';
            setTimeout(function() {
                alert.remove();
            }, 500);
        }, 5000);
    });
});
