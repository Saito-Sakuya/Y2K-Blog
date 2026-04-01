import { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import api from '../api/client';
import axios from 'axios';

export default function Login() {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  
  // Captcha state
  const [needsCaptcha, setNeedsCaptcha] = useState(false);
  const [captchaQuestion, setCaptchaQuestion] = useState('');
  const [captchaToken, setCaptchaToken] = useState('');
  const [captchaAnswer, setCaptchaAnswer] = useState('');

  // Block state
  const [blocked, setBlocked] = useState(false);
  const [retrySeconds, setRetrySeconds] = useState(0);

  const { login } = useAuth();
  const navigate = useNavigate();

  // Check login status (captcha/block)
  const checkLoginStatus = useCallback(async () => {
    try {
      const res = await api.get('/admin/login/status');
      const data = res.data;

      if (data.blocked) {
        setBlocked(true);
        setRetrySeconds(data.retryAfterSeconds || 0);
        setNeedsCaptcha(false);
      } else {
        setBlocked(false);
        setRetrySeconds(0);
        if (data.needsCaptcha) {
          setNeedsCaptcha(true);
          await fetchCaptcha();
        } else {
          setNeedsCaptcha(false);
        }
      }
    } catch {
      // If status endpoint fails, proceed without captcha
    }
  }, []);

  const fetchCaptcha = async () => {
    try {
      const res = await api.get('/admin/captcha');
      setCaptchaToken(res.data.token);
      setCaptchaQuestion(res.data.question);
      setCaptchaAnswer('');
    } catch {
      setCaptchaQuestion('Failed to load captcha');
    }
  };

  useEffect(() => {
    checkLoginStatus();
  }, [checkLoginStatus]);

  // Countdown timer for blocked state
  useEffect(() => {
    if (!blocked || retrySeconds <= 0) return;
    const timer = setInterval(() => {
      setRetrySeconds((prev) => {
        if (prev <= 1) {
          clearInterval(timer);
          setBlocked(false);
          checkLoginStatus();
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
    return () => clearInterval(timer);
  }, [blocked, retrySeconds, checkLoginStatus]);

  const formatCountdown = (secs: number) => {
    const m = Math.floor(secs / 60);
    const s = secs % 60;
    return `${m}:${s.toString().padStart(2, '0')}`;
  };

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setIsLoading(true);

    try {
      const payload: any = { username, password };
      if (needsCaptcha) {
        payload.captchaToken = captchaToken;
        payload.captchaAnswer = parseInt(captchaAnswer, 10);
      }

      const response = await api.post('/admin/login', payload);
      
      const { token } = response.data;
      if (token) {
        login(token, username);
        navigate('/dashboard');
      } else {
        setError('Login failed: No token received.');
      }
    } catch (err) {
      if (axios.isAxiosError(err) && err.response) {
        const data = err.response.data;
        setError(data.error || 'Invalid credentials.');

        // Handle specific error codes
        if (data.code === 'CAPTCHA_REQUIRED') {
          setNeedsCaptcha(true);
          await fetchCaptcha();
        } else if (data.code === 'CAPTCHA_INVALID') {
          // Refresh captcha on wrong answer
          await fetchCaptcha();
        } else if (data.code === 'IP_BLOCKED') {
          setBlocked(true);
          setRetrySeconds(data.retryAfterSeconds || 900);
          setNeedsCaptcha(false);
        } else {
          // Re-check status after any failed login
          await checkLoginStatus();
        }
      } else {
        setError('Network error. Is the API running?');
      }
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', width: '100%' }}>
      <div className="window y2k-window" style={{ width: 380 }}>
        <div className="title-bar">
          <div className="title-bar-text">🔑 Auth - Y2K Admin</div>
          <div className="title-bar-controls">
            <button aria-label="Minimize" />
            <button aria-label="Maximize" disabled />
            <button aria-label="Close" />
          </div>
        </div>
        
        <div className="window-body">
          {blocked ? (
            /* Blocked View */
            <div style={{ textAlign: 'center', padding: '16px 0' }}>
              <p style={{ fontSize: 32, marginBottom: 8 }}>🚫</p>
              <p style={{ fontWeight: 'bold', marginBottom: 8, color: 'red' }}>Access Temporarily Blocked</p>
              <p style={{ fontSize: 13, marginBottom: 16 }}>
                Too many failed login attempts. Please wait before trying again.
              </p>
              <div style={{ 
                fontFamily: 'monospace', fontSize: 28, fontWeight: 'bold', 
                padding: '12px', backgroundColor: '#c0c0c0', 
                border: '2px inset #fff', display: 'inline-block',
                minWidth: 100
              }}>
                {formatCountdown(retrySeconds)}
              </div>
              <p style={{ fontSize: 11, marginTop: 12, color: '#666' }}>
                IP will be unblocked automatically
              </p>
            </div>
          ) : (
            /* Login Form */
            <>
              <p style={{ textAlign: 'center', marginBottom: 16 }}>Please enter your credentials.</p>
              
              <form onSubmit={handleLogin}>
                <div className="field-row-stacked" style={{ marginBottom: 8 }}>
                  <label htmlFor="username">Username</label>
                  <input
                    id="username"
                    type="text"
                    value={username}
                    onChange={(e) => setUsername(e.target.value)}
                    required
                  />
                </div>
                
                <div className="field-row-stacked" style={{ marginBottom: 8 }}>
                  <label htmlFor="password">Password</label>
                  <input
                    id="password"
                    type="password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    required
                  />
                </div>

                {/* Captcha Section */}
                {needsCaptcha && (
                  <fieldset style={{ marginBottom: 8 }}>
                    <legend>Verification</legend>
                    <div style={{ 
                      fontFamily: 'monospace', fontSize: 18, fontWeight: 'bold',
                      padding: '8px 12px', marginBottom: 8,
                      backgroundColor: '#c0c0c0', border: '2px inset #fff',
                      textAlign: 'center', letterSpacing: 2
                    }}>
                      {captchaQuestion || 'Loading...'}
                    </div>
                    <div className="field-row-stacked">
                      <label htmlFor="captcha-answer">Your Answer</label>
                      <input
                        id="captcha-answer"
                        type="number"
                        value={captchaAnswer}
                        onChange={(e) => setCaptchaAnswer(e.target.value)}
                        required
                        placeholder="Enter the answer"
                        style={{ fontFamily: 'monospace' }}
                      />
                    </div>
                    <button type="button" onClick={fetchCaptcha} style={{ marginTop: 4, fontSize: 11 }}>
                      🔄 New Question
                    </button>
                  </fieldset>
                )}
                
                {error && (
                  <p style={{ color: 'red', fontSize: 12, marginBottom: 8 }}>❌ {error}</p>
                )}
                
                <div style={{ display: 'flex', justifyContent: 'center', gap: 8, marginTop: 16 }}>
                  <button disabled={isLoading} type="submit" style={{ width: 100 }}>
                    {isLoading ? 'Wait...' : 'OK'}
                  </button>
                  <button type="button" onClick={() => { setUsername(''); setPassword(''); setCaptchaAnswer(''); }} style={{ width: 100 }}>
                    Cancel
                  </button>
                </div>
              </form>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
