import base64
import json

import requests
from prompt_toolkit.shortcuts import (
    checkboxlist_dialog,
    input_dialog,
    message_dialog,
    radiolist_dialog,
)

from src.crypto import encrypt_data, generate_keys, sign_message
from src.network import get_possible_pubs, send_transaction
from src.storage import save_to_file
from src.transactions import Transaction, TransactionMessage

type KeyPair = tuple[str, str]
PK = 1
SK = 0


def view_key(key: str):
    return (
        key.replace("-----BEGIN PUBLIC KEY-----", "")
        .replace("-----END PUBLIC KEY-----", "")
        .replace("\n", "")
        .strip()
    )


class App:
    def __init__(self, password, keys, storage_path, node_adr=None) -> None:
        self.password: str = password
        self.keys: set[KeyPair] = keys
        self.storage_path = storage_path
        self.node_adr: None | str = node_adr

        self.chosen_id: KeyPair | None = None
        self.coins: float | None = None

    def run(self):
        while True:
            if self._render():
                break

    def _render(self) -> None | str:
        key_options = [(pair, pair[1]) for pair in self.keys]
        radio_options = [
            ("add", "Generate new pair"),
            ("list", "Chose identity"),
            ("delete", "Delete selected pairs"),
            ("connect", "Connect to node"),
            ("transaction", "Send transaction"),
            ("exit", "Exit"),
        ]
        text = f"""Connected to node: {self.node_adr}
Identity: {'Chose identity' if not self.chosen_id else view_key(self.chosen_id[PK])[:24] + '...'}
Smolcoins: {'Chose identity' if not self.coins else self.coins}
        """
        choice = radiolist_dialog(title="Wallet", text=text, values=radio_options).run()

        if choice == "add":
            self._add_new_keys()

        elif choice == "list":
            self.chosen_id = radiolist_dialog(
                title="Chose identity", values=key_options
            ).run()

        elif choice == "delete":
            delete_choice = checkboxlist_dialog(
                title="Delete keys", values=key_options
            ).run()
            for pair in delete_choice:
                self._delete_key(pair)

        elif choice == "connect":
            node_adr = input_dialog(text="Provide node address").run()
            try:
                requests.get(url=f"http://{node_adr}/")
            except:
                node_adr = None

        elif choice == "transaction":
            if not self.node_adr:
                message_dialog(title="Error", text="Node address is not set.").run()
                return
            if not self.chosen_id:
                message_dialog(title="Error", text="Identity is not chosen").run()
                return

            tx = self._render_tx()
            if not tx:
                return
            tx_str = json.dumps(tx, sort_keys=True)
            signature_str = base64.b64encode(
                sign_message(self.chosen_id[SK], tx_str)
            ).decode("utf-8")
            send_transaction(
                self.node_adr,
                TransactionMessage(tx_str, signature_str, tx.sender),
            )

        elif choice == "exit":
            return "EXIT"

    def _render_tx(self) -> None | Transaction:
        assert isinstance(self.node_adr, str)
        assert self.chosen_id is not None

        tx = Transaction()
        tx.difficulty = 5
        tx.sender = self.chosen_id[PK]

        while True:
            transaction_options = [
                ("reciever", f"Reciever: {tx.reciever}"),
                ("ammount", f"Ammount: {tx.ammount}"),
                ("difficulty", f"Difficulty: {tx.difficulty}"),
                ("send", "Send"),
                ("cancel", "Cancel transaction"),
            ]
            tx_choice = radiolist_dialog(
                title="New Transaction",
                values=transaction_options,
                text=f"Sender: {view_key(self.chosen_id[PK])[:24] + '...'}",
            ).run()

            if tx_choice == "reciever":
                reciever_choice = get_possible_pubs(self.node_adr)
                tx.reciever = radiolist_dialog(
                    title="Chose reciever",
                    values=[(x, x) for x in reciever_choice],
                ).run()
            elif tx_choice == "ammount":
                tx.ammount = float(input_dialog(text="Set ammount").run())
            elif tx_choice == "difficulty":
                tx.difficulty = int(input_dialog(text="Set difficulty").run())
            elif tx_choice == "send":
                break
            elif tx_choice == "cancel":
                return
        return tx

    def _save_keys(self):
        data = []
        for pair in self.keys:
            encrypted = encrypt_data(self.password, json.dumps(pair))
            data.append(encrypted)
        save_to_file(self.storage_path, data)

    def _add_new_keys(self):
        pair = generate_keys()
        self.keys.add(pair)
        self._save_keys()

    def _delete_key(self, pair: tuple[str, str]):
        self.keys.remove(pair)
        self._save_keys()
