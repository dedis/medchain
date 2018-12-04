CREATE TABLE ClientTransaction(
  uid TEXT PRIMARY KEY,
  client_transaction TEXT,
  description TEXT,
  status TEXT
);

CREATE TABLE SignatureStatus(
  transaction_uid TEXT,
  signer_identity TEXT,
  status TEXT DEFAULT "WAITING",
  FOREIGN KEY(transaction_uid) REFERENCES ClientTransaction(uid),
  PRIMARY KEY(transaction_uid, signer_identity)
);
