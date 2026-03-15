# achan.moe

anonymous imageboard. accounts are random 16-digit IDs with BIP39 seed phrase recovery. no email, no phone number.

## stats

total lines of go: [ 22735 ] at 2026-03-15

## why e2e

the site sits behind cloudflare, which terminates TLS and can read traffic in plaintext. credentials, posts, and uploaded media are encrypted in the browser before they reach the server. cloudflare only sees ciphertext.

the crypto is post-quantum (ML-KEM-768 for key exchange, ML-DSA-65 for signing, AES-256-GCM for symmetric encryption) to handle both current and future threats.

## what this repo is for

out-of-band verification and MITM mitigation for the e2e signing key.

`e2e-signing-key.json` contains the current signing key fingerprint and SHA-256 hashes of client JS assets. updated automatically on key rotation (daily 05:00 UTC).

the client fetches this file from both github and gitlab and compares it to the key served by the site. if anything disagrees, a MITM warning is shown and e2e is disabled. an attacker would need to compromise three independent TLS chains (cloudflare, github, gitlab) simultaneously.

while the native client verifies the signing key against these out-of-band sources, the JS itself is still delivered through cloudflare and could be tampered with. [`achan-integrity.user.js`](achan-integrity.user.js) is a tampermonkey script that runs outside that delivery chain, hashing the loaded JS against the published values here. it catches what in-page verification can't.

## docs

planned for `docs/`. not there yet.

## license

proprietary. all rights reserved.
