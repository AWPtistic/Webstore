Clearly we don't want to store passwords in plaintext. Do some research and write a short paragraph about what you would do to make the passwords more security. NOTE: All of thespe questions refer to system design, NOT your personal practice.
        It is a bad practice to store passwords in plain text. 
        Use the appropriate algorithms for password encryption such as bcrypt or Argon2, which are specifically designed for storing passwords. 
        These algorithms work with salting, which helps shelter the password with a specific integer value prior to hashing, 
        making it impossible to employ the pre-computed hash table like rainbow tables. 
        Always keep only the salted hash in the database, otherwise, even if the database gets cracked, passwords will remain safe.

What are some good/not so good ways to deal with forgotten passwords? List at least one of each.
        Good: Time-limited reset link via email.
        Bad: Emailing plaintext passwords.
What are some things to consider when implementing "Remember me" functionality? List at least two.
        Use tokens, secure cookies, expiration, and revocation mechanisms.
What are some best practices for cookies? List at least two.
        Set HttpOnly, Secure, and SameSite attributes
What is https?
    "Hypertext transfer protocol secure (HTTPS) is the secure version of HTTP, which is the primary protocol used to send data between a web browser and a website."
        -google