import base64
import os

from cryptography.hazmat.primitives import hashes, padding, serialization
from cryptography.hazmat.primitives.asymmetric import ec
from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes
from cryptography.hazmat.primitives.kdf.pbkdf2 import PBKDF2HMAC

### CONSTANTS
DKEY_LEN = 32
PBKDF2_ITERS = 100_000
SALT_LEN = 16
IV_LEN = 16


def generate_keys() -> tuple[str, str]:
    private_key = ec.generate_private_key(ec.SECP256R1())
    public_key = private_key.public_key()

    prv_pem = private_key.private_bytes(
        encoding=serialization.Encoding.PEM,
        format=serialization.PrivateFormat.PKCS8,
        encryption_algorithm=serialization.NoEncryption(),
    ).decode("utf-8")

    pub_pem = public_key.public_bytes(
        encoding=serialization.Encoding.PEM,
        format=serialization.PublicFormat.SubjectPublicKeyInfo,
    ).decode("utf-8")

    return prv_pem, pub_pem


def _derive_key(password: bytes, salt: bytes) -> bytes:
    kdf = PBKDF2HMAC(
        algorithm=hashes.SHA256(), length=DKEY_LEN, iterations=PBKDF2_ITERS, salt=salt
    )
    return kdf.derive(password)


def encrypt_data(password: str, data: str) -> dict:
    data_bytes = data.encode("utf-8")
    salt = os.urandom(SALT_LEN)

    key = _derive_key(password.encode("utf-8"), salt)
    iv = os.urandom(IV_LEN)

    aes = algorithms.AES256(key)
    cipher = Cipher(aes, modes.CBC(iv))
    encryptor = cipher.encryptor()

    padder = padding.PKCS7(aes.block_size).padder()
    padded_data = padder.update(data_bytes) + padder.finalize()

    ciphertext = encryptor.update(padded_data) + encryptor.finalize()
    return {
        "salt": base64.b64encode(salt).decode("utf-8"),
        "iv": base64.b64encode(iv).decode("utf-8"),
        "encoded": base64.b64encode(ciphertext).decode("utf-8"),
    }


def decrypt_data(password: str, encoded_data: dict) -> str:
    key = _derive_key(password.encode("utf-8"), base64.b64decode(encoded_data["salt"]))
    aes = algorithms.AES256(key)
    cipher = Cipher(aes, modes.CBC(base64.b64decode(encoded_data["iv"])))
    decryptor = cipher.decryptor()

    padded_data = decryptor.update(base64.b64decode(encoded_data["encoded"]))

    unpadder = padding.PKCS7(aes.block_size).unpadder()
    plain_text = unpadder.update(padded_data) + unpadder.finalize()
    return plain_text.decode("utf-8")


def sign(private_key_pem: str, message: str) -> bytes:
    private_key = serialization.load_pem_private_key(
        private_key_pem.encode("utf-8"),
        password=None,
    )
    assert isinstance(private_key, ec.EllipticCurvePrivateKey)

    message_bytes = message.encode("utf-8")

    # ECDSA (Elliptic Curve Digital Signature Algorithm)
    # with SHA-256 as the hash function.
    return private_key.sign(message_bytes, ec.ECDSA(hashes.SHA256()))
