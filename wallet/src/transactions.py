from dataclasses import dataclass


@dataclass
class Transaction:
    sender: str
    reciever: str
    ammount: float
    timestamp: int
    difficulty: int

    def __init__(self) -> None:
        self.sender = ""
        self.reciever = ""
        self.ammount = 0.0
        self.timestamp = 0
        self.difficulty = 0


@dataclass
class TransactionMessage:
    transaction: str
    signature: str
    pub_key: str
