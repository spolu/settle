/*
 * Fact Smart Contract V0001
 */

contract owned {
    function owned() {
        owner = msg.sender;
    }
    modifier onlyowner() {
        if (msg.sender == owner)
            _
    }
    address owner;
}

contract mortal is owned {
    function kill() {
        if (msg.sender == owner) suicide(owner);
    }
}

contract Fact is owned, mortal {
    bytes8 constant VERSION = 0x0001;

    // 0x010* is reserved for legal information.
    bytes8 constant T_FULL_NAME             = 0x0101;
    bytes8 constant T_ENTITY_TYPE           = 0x0102;
    bytes8 constant T_DATE_OF_BIRTH         = 0x0103;
    bytes8 constant T_DATE_OF_CREATION      = 0x0104;
    bytes8 constant T_DATE_OF_INCORPORATION = 0x0105;
    // 0x020* is reserved for contact information.
    bytes8 constant T_EMAIL                 = 0x0201;
    bytes8 constant T_PHONE                 = 0x0202;
    bytes8 constant T_URL                   = 0x0203;
    // 0x030* is reserved for address information.
    bytes8 constant T_ADDRESS_LINE1         = 0x0301;
    bytes8 constant T_ADDRESS_LINE2         = 0x0302;
    bytes8 constant T_ADDRESS_CITY          = 0x0303;
    bytes8 constant T_ADDRESS_POSTAL_CODE   = 0x0304;
    bytes8 constant T_STATE                 = 0x0305;
    bytes8 constant T_COUNTRY               = 0x0306;
    // 0x040* is reserved for official document IDs.
    bytes8 constant T_TAX_ID                = 0x0401;
    bytes8 constant T_PASSPORT_NUMBER       = 0x0402;
    bytes8 constant T_DRIVER_LICENSE_ID     = 0x0403;
    // 0x050* is reserved for financial information.
    bytes8 constant T_BANK_ACCOUNT_NUMBER   = 0x0501;
    bytes8 constant T_CARD_NUMBER           = 0x0502;
    bytes8 constant T_BITCOIN_ADDRESS       = 0x0502;
    // 0x060* is reserved for social profiles.
    bytes8 constant T_FACEBOOK_ID           = 0x0601;
    bytes8 constant T_TWITTER_HANDLE        = 0x0602;
    bytes8 constant T_REDDIT_HANDLE         = 0x0603;
    bytes8 constant T_GITHUB_HANDLE         = 0x0604;

    // 0x0*** is reserved for future uses.
    // 0x9*** can be used for custom application specific fact types.

    bytes8 public factType;
    mapping (address => bytes32) public assertions;

    event FactCreated(address indexed owner, bytes8 factType, bytes32 factHash);
    event FactAsserted(address indexed authority, bytes32 factHash);
    event FactRevoked(address indexed authority);

    // Initializes a new Fact by providing a `_factType` (a fact type value
    // that should be coherent with the conventions above) and a `_hash =
    // sha3(owner, name)`.
    function Fact(
        bytes8 _factType,
        bytes32 _hash
    ) {
        owner = msg.sender;
        factType = _factType;
        assertions[msg.sender] = _hash;

        FactCreated(owner, factType, _hash);
    }

    // Asserts a Fact by providing `hash = sha3(authority, value)`. The validity
    // of the hash is not enforced by the contract. Instead, when a contract or
    // an external agent with access to the value underlying the Fact, attempts
    // to check if an authority asserts a Fact, they should compute `hash =
    // sha3(authority, value)` and call `check(authority, hash)`. The provided
    // hash will be verified against the hash provided by the authority at
    // assertion time.
    function assert(bytes32 hash) public {
        assertions[msg.sender] = hash;

        FactAsserted(msg.sender, hash);
    }

    // Revokes a fact assertion. If the caller of this function is the fact
    // owner, the contract self-destructs.
    function revoke() public {
       assertions[msg.sender] = bytes32(0);
       FactRevoked(owner);

       kill();
    }

    // Checks whether the provided authority asserts the Fact for the hash
    // (hence value) provided.
    function check(address authority, bytes32 hash) public returns (bool) {
        return assertions[authority] == hash;
    }

    function() {
        throw;
    }
}
