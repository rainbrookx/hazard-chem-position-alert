/**
 * RSA 非对称加密工具 — 前端登录加密。
 * 使用 Web Crypto API SubtleCrypto，零外部依赖。
 */

/**
 * 使用 RSA 公钥（PEM 格式）加密文本。
 * @returns base64 编码的密文
 */
export async function encryptWithPublicKey(
  plaintext: string,
  publicKeyPem: string,
): Promise<string> {
  // 将 PEM 公钥转成 CryptoKey
  const pemHeader = '-----BEGIN PUBLIC KEY-----'
  const pemFooter = '-----END PUBLIC KEY-----'
  const pemContents = publicKeyPem.replace(pemHeader, '').replace(pemFooter, '').replace(/\s/g, '')

  const binaryDer = Uint8Array.from(atob(pemContents), (c) => c.charCodeAt(0))

  const cryptoKey = await crypto.subtle.importKey(
    'spki',
    binaryDer,
    { name: 'RSA-OAEP', hash: 'SHA-256' },
    false,
    ['encrypt'],
  )

  const encoded = new TextEncoder().encode(plaintext)
  const encrypted = await crypto.subtle.encrypt({ name: 'RSA-OAEP' }, cryptoKey, encoded)

  return btoa(String.fromCharCode(...new Uint8Array(encrypted)))
}

/**
 * 加密用户名和密码，返回 base64 编码的密文对。
 */
export async function encryptCredentials(
  username: string,
  password: string,
  publicKeyPem: string,
): Promise<{ username: string; password: string }> {
  const [encUsername, encPassword] = await Promise.all([
    encryptWithPublicKey(username, publicKeyPem),
    encryptWithPublicKey(password, publicKeyPem),
  ])
  return { username: encUsername, password: encPassword }
}
