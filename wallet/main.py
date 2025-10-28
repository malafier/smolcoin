import json

from prompt_toolkit import prompt
from prompt_toolkit.shortcuts import message_dialog, radiolist_dialog

from src.crypto import decrypt_data, encrypt_data, generate_keys
from src.storage import load_from_file, save_to_file

### Constants
KEY_STORAGE_PATH = "./key_store.json"


### Wallet State
keys: set[tuple[str, str]] = set()


def add_new_keys(password: str):
    pair = generate_keys()
    keys.add(pair)

    data = []
    for pair in keys:
        encrypted = encrypt_data(password, json.dumps(pair))
        data.append(encrypted)
    save_to_file(KEY_STORAGE_PATH, data)


def main():
    password = prompt("Enter password: ", is_password=True)

    loaded_data = load_from_file(KEY_STORAGE_PATH)
    decrypted = [decrypt_data(password, x) for x in loaded_data]
    for key_pair in decrypted:
        keys.add(tuple(json.loads(key_pair)))

    while True:
        radio_options = [
            ("add", "Generate new pair"),
            ("list", "List all pairs"),
            ("exit", "Exit"),
        ]

        choice = radiolist_dialog(title="Wallet", values=radio_options).run()

        if choice == "add":
            add_new_keys(password)
        elif choice == "list":
            displayed_text = ""
            for pair in keys:
                displayed_text += f"{pair[0]}\n{pair[1]}\n\n"
            message_dialog(title="Keys", text=displayed_text).run()
        elif choice == "exit":
            break


if __name__ == "__main__":
    main()
