import base64
import json
from dataclasses import dataclass

from src.crypto import sign


@dataclass
class Transaction:
    sender: str
    reciever: str
    ammount: float
    timestamp: int
    difficulty: int
    signature: str

    def __init__(self) -> None:
        self.sender = ""
        self.reciever = ""
        self.ammount = 0.0
        self.timestamp = 0
        self.difficulty = 0

    def sign(self, sk: str):
        tx = {
            "sender": self.sender,
            "reciever": self.reciever,
            "ammount": self.ammount,
            "timestamp": self.timestamp,
            "difficulty": self.difficulty,
        }
        tx_str = json.dumps(tx, sort_keys=True)
        self.signature = sign(sk, tx_str).hex()
