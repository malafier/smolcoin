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

    def run(self):
        while True:
            if self._render():
                break

    def _render(self) -> None | str:
        key_options = [(pair, pair[1]) for pair in self.keys]
        radio_options = [
            ("add", "Generate new pair"),
            ("list", "List all pairs"),
            ("delete", "Delete selected pairs"),
            ("connect", "Connect to node"),
            ("transaction", "Send transaction"),
            ("exit", "Exit"),
        ]
        text = f"Connected to node: {self.node_adr}"
        choice = radiolist_dialog(title="Wallet", text=text, values=radio_options).run()

        if choice == "add":
            self._add_new_keys()

        elif choice == "list":
            displayed_text = ""
            for pair in self.keys:
                displayed_text += f"{pair[0]}\n{pair[1]}\n\n"
            message_dialog(title="Keys", text=displayed_text).run()

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

            maybe = self._render_tx(key_options)
            if not maybe:
                return
            tx, key = maybe
            tx_str = json.dumps(tx, sort_keys=True)
            signature_str = base64.b64encode(sign_message(key, tx_str)).decode("utf-8")
            send_transaction(
                self.node_adr,
                TransactionMessage(tx_str, signature_str, tx.sender),
            )

        elif choice == "exit":
            return "EXIT"

    def _render_tx(
        self, key_options: list[tuple[KeyPair, str]]
    ) -> None | tuple[Transaction, str]:
        assert isinstance(self.node_adr, str)
        tx = Transaction()
        tx.difficulty = 5
        key_chosen = None

        while True:
            transaction_options = [
                (
                    "sender",
                    f"Sender: {key_chosen if not key_chosen else view_key(key_chosen[PK])[:16] + '...'}",
                ),
                ("reciever", f"Reciever: {tx.reciever}"),
                ("ammount", f"Ammount: {tx.ammount}"),
                ("difficulty", f"Difficulty: {tx.difficulty}"),
                ("send", "Send"),
                ("cancel", "Cancel transaction"),
            ]

            tx_choice = radiolist_dialog(
                title="New Transaction", values=transaction_options
            ).run()
            if tx_choice == "sender":
                key_chosen = radiolist_dialog(
                    title="Chose identity", values=key_options
                ).run()
            elif tx_choice == "reciever":
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

        if not key_chosen:
            return
        tx.sender = key_chosen[PK]
        return tx, key_chosen[SK]

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
