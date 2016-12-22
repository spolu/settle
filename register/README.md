# Register

The register service can be used in conjonction with a mint to allow public
registration on it. This is what powers registration of the `settle.network`
mint.

Registration is made available from the `settle` command line and consists in:
- capturing the email of the user in the command line
- submitting a registration request to the register service
- sending a credentials link to the user over email

Once the user follows the credentials link, we verify that he is human and
display his credentials to them.

It can be used by other mints, but is not mandatory to run a mint (registration
of users is left out of the specification of a mint).
