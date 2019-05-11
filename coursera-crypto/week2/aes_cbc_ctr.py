from Crypto.Cipher import AES
from Crypto.Util import Counter

def decrypt_cbc_use_lib(key, cipher_text):
    iv = bytes.fromhex(cipher_text[0:AES.block_size*2])
    cipher = AES.new(bytes.fromhex(key), AES.MODE_CBC, iv)
    msg = cipher.decrypt(bytes.fromhex(cipher_text[AES.block_size*2:]))
    return msg.decode()

def ctr(iv):
    counter_val = 0 
    def new_ctr():
        nonlocal counter_val
        nxt = counter_val + int.from_bytes(iv, byteorder='big')
        counter_val += 1 
        return nxt.to_bytes(AES.block_size, byteorder='big')
    return new_ctr
        
def decrypt_ctr_use_lib(key, cipher_text):
    iv = bytes.fromhex(cipher_text[0:AES.block_size*2])
    cipher = AES.new(bytes.fromhex(key), AES.MODE_CTR, iv, counter=ctr(iv))
    msg = cipher.decrypt(bytes.fromhex(cipher_text[AES.block_size*2:]))
    counter_val = 0
    return msg.decode()

# Q1
key1 = "140b41b22a29beb4061bda66b6747e14"
cipher_text1 = "4ca00ff4c898d61e1edbf1800618fb2828a226d160dad07883d04e008a7897ee2e4b7465d5290d0c0e6c6822236e1daafb94ffe0c5da05d9476be028ad7c1d81"

# Q2
key2 = "140b41b22a29beb4061bda66b6747e14"
cipher_text2 = "5b68629feb8606f9a6667670b75b38a5b4832d0f26e1ab7da33249de7d4afc48e713ac646ace36e872ad5fb8a512428a6e21364b0c374df45503473c5242a253"

# Q3
key3 = "36f18357be4dbd77f050515c73fcf9f2"
cipher_text3 = "69dda8455c7dd4254bf353b773304eec0ec7702330098ce7f7520d1cbbb20fc388d1b0adb5054dbd7370849dbf0b88d393f252e764f1f5f7ad97ef79d59ce29f5f51eeca32eabedd9afa9329"

# Q4
key4 = "36f18357be4dbd77f050515c73fcf9f2"
cipher_text4 = "770b80259ec33beb2561358a9f2dc617e46218c0a53cbeca695ae45faa8952aa0e311bde9d4e01726d3184c34451"

print(decrypt_cbc_use_lib(key1, cipher_text1))
print(decrypt_cbc_use_lib(key2, cipher_text2))

print(decrypt_ctr_use_lib(key3, cipher_text3))
print(decrypt_ctr_use_lib(key4, cipher_text4))



