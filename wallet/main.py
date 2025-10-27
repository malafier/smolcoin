import json

from prompt_toolkit import prompt

from src.crypto import decrypt_data, encrypt_data, generate_keys
from src.storage import load_from_file, save_to_file

### Wallet State
keys = set()
password: str | None = None


def main():
    password = prompt("Enter password: ", is_password=True)
    prv, pub = generate_keys()
    pair = [{"private_key": prv, "public_key": pub}]

    data = json.dumps(pair)
    print(data)
    encrypted = encrypt_data(password, data)
    print(encrypted)
    save_to_file("storage.json", [encrypted])

    ldata = load_from_file("storage.json")
    print(ldata)

    decrypted = decrypt_data(password, ldata[0])
    print(decrypted)

    print(decrypted == data)


if __name__ == "__main__":
    main()
