from dataclasses import dataclass


@dataclass
class Transaction:
    sender: str
    reciever: str
    ammount: float
    timestamp: int
    difficulty: int


@dataclass
class TransactionMessage:
    transaction: str
    signature: str
    pub_key: str
