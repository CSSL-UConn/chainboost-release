Scalar represents a scalar value by which a Point (group element) may be encrypted to produce another Point. => a scalar multiplier in elliptic curve group

Parameter Selection:
λ: security parameter; 
Typically, λ=80. 
P should be a 2λ-bit prime, and the curve should be chosen so that the discrete logarithm is 2λ-secure. 
For values of λ up to 128, Barreto–Naehrig curves are the right choice

Package bn256: implements the Optimal Ate pairing over a 256-bit Barreto-Naehrig curve as described in <http://cryptojedi.org/papers/dclxvi-20100714.pdf>.
This package previously claimed to operate at a 128-bit security level. However, recent improvements in attacks mean that is no longer true. See <https://moderncrypto.org/mail-archive/curves/2016/000740.html>.
Package bn256 from kyber library is used in blscosi module for bls scheme.


There is another kind of signing in por implementation (what suite it should use?) now its using the same suite as BaseDFS is using

PoR implementation:
const s = 10 // number of sectors in eac block (sys. par.)
// Each sector is one element of Zp, and there are s sectors per block.
// If the processed file is b bits long,then there are n = [b/s lg p] blocks.
const n = 10 // number of blocks
const l = 5  //size of query set (i<n)
var suite = pairing.NewSuiteBn256()
<https://www.youtube.com/watch?v=8_9ONpyRZEI>



Cryptographic Parameters :

For BLS signatures we use the bilinear bn256 (New  software  speed records  for  cryptographic  pairings) group of elliptic curves. 
At the 128-bit security level, a nearly optimal choice for a pairing-friendly curve is a Barreto-Naehrig (BN) curve over a prime field of size roughly 256 bits with embedding degree k = 12. 
bits of security: 128 bits / 124 bits. 
New  software  speed records  for  cryptographic  pairings:  a constant-time implementation of an optimal ate pairing on a BN curve over a prime field Fp of size 257 bits. 
The prime p is given by the BN polynomial parametrization p = 36u4+36u3+24u2+6u+1, where u = v3 and v = 1966080. The curve equation is E : y2 = x3 + 17.
Compact por: 
"Let lambda be the security parameter; typically, lambda=80. For the scheme with public verification,p should be a 2 lambda-bit prime, and the curve should be chosen so that discrete logarithm is 2 lambda-secure. For values of lambda up to 128, Barreto–Naehrig curves are the right choice"
