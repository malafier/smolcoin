import json
from dataclasses import dataclass

from src.crypto import sign


@dataclass
class Transaction:
    sender: str
    reciever: str
    ammount: float
    timestamp: int
    signature: str

    def __init__(self) -> None:
        self.sender = ""
        self.reciever = ""
        self.ammount = 0.0
        self.timestamp = 0

    def sign(self, sk: str):
        tx = {
            "sender": self.sender,
            "reciever": self.reciever,
            "ammount": int(self.ammount) if self.ammount.is_integer() else self.ammount,
            "timestamp": self.timestamp,
        }
        tx_str = json.dumps(tx, separators=(",", ":"))
        print("msg:\n", tx_str)
        self.signature = sign(sk, tx_str).hex()
        print("sig:\n", self.signature)
