# PoR implementation #:
This is the implementation of **the scheme with public verifiability** proposed in **Compact Proofs of Retrievability** by Shacham and Waters: an efficient public-key PoR (from CDH in bilinear groups) using the homomorphic properties of BLS signatures <https://eprint.iacr.org/2008/073.pdf>
- public verifiability
- efficient 
- still rely on a private-key client to privately preprocess the file [^6]
- The client’s query and server’s response are both extremely short: 20 bytes and 40 bytes, respectively, at the 80-bit security level.

## Cryptographic Parameters ##:

- For BLS signatures we use the bilinear bn256 (New  software  speed records  for  cryptographic  pairings [^5]) group of elliptic curves. 
- At the 128-bit security level, a nearly optimal choice for a pairing-friendly curve is a Barreto-Naehrig (BN) curve over a prime field of size roughly 256 bits with embedding degree k = 12. 
- bits of security: 128 bits / 124 bits. 
The prime p is given by the BN polynomial parametrization p = 36u4+36u3+24u2+6u+1, where u = v3 and v = 1966080. 
The curve equation is E : y2 = x3 + 17.

## Parameter Selection ##

| Param Name    | Initial value           | Description                                    |
| :-----------: | :---------------------: | :--------------------------------------------: |
| const s       | 10                      | number of sectors in eac block (sys. par.) [^3]|
| const n       | 10                      |   number of blocks                             |
| const l       | 5                       |    size of query set (i<n)                     |
|var λ          | Typically, λ=80 [^4]    | security parameter                             |
|var suite      | pairing.NewSuiteBn256() | [^1] Also see [^2] and [^4]                         |


----------------------------------
- Scalar represents a scalar value by which a Point (group element) may be encrypted to produce another Point => a scalar multiplier in elliptic curve group



<!--FootNote-->
[^1]: Package bn256: implements the Optimal Ate pairing over a 256-bit Barreto-Naehrig curve as described in <http://cryptojedi.org/papers/dclxvi-20100714.pdf>. This package previously claimed to operate at a 128-bit security level.
[^2]: <https://www.youtube.com/watch?v=8_9ONpyRZEI>
[^3]: Each sector is one element of Zp, and there are s sectors per block. If the processed file is b bits long,then there are n = [b/s lg p] blocks.
[^4]: P should be a 2λ-bit prime, and the curve should be chosen so that the discrete logarithm is 2λ-secure. For values of λ up to 128, **Barreto–Naehrig curves** are the right choice.
[^5]: a constant-time implementation of an optimal ate pairing on a BN curve over a prime field Fp of size 257 bits
[^6]: Disadv: If the prover and client collude then the proof is meaningless to the verifier!
<!--FootNote-->
