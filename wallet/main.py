import json

from prompt_toolkit import prompt

from src.crypto import decrypt_data
from src.storage import load_from_file
from src.ui import App

### Wallet State
keys: set[tuple[str, str]] = set()
node_adr: str | None = "localhost:3001"


def main():
    login = prompt("Enter login: ")
    password = prompt("Enter password: ", is_password=True)
    path = f"./{login}_store.json"

    loaded_data = load_from_file(path)
    try:
        decrypted = [decrypt_data(password, x) for x in loaded_data]
        for key_pair in decrypted:
            keys.add(tuple(json.loads(key_pair)))
    except:
        print("Failed to decrypt file.")
        exit(1)

    print(keys)
    app = App(password, keys, path, node_adr)
    app.run()


if __name__ == "__main__":
    main()
