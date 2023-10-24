// SPDX-License-Identifier: AGPL-3.0

pragma solidity 0.8.17;

import { ReentrancyGuardUpgradeable } from "@openzeppelin/contracts-upgradeable/security/ReentrancyGuardUpgradeable.sol";
import { OwnableUpgradeable } from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import { Address } from "@openzeppelin/contracts/utils/Address.sol";
import { SafeERC20Upgradeable } from "@openzeppelin/contracts-upgradeable/token/ERC20/utils/SafeERC20Upgradeable.sol";
import "@openzeppelin/contracts-upgradeable/token/ERC20/extensions/IERC20MetadataUpgradeable.sol";

import { IOpenRegistrar } from "./interfaces/IOpenRegistrar.sol";
import { IOpsideSlots } from "./interfaces/IOpsideSlots.sol";
import { Request } from "./util/Structs.sol";

contract OpenRegistrar is IOpenRegistrar, ReentrancyGuardUpgradeable, OwnableUpgradeable {
    using SafeERC20Upgradeable for IERC20Upgradeable;

    event NewRegistration(uint256 indexed regId, uint256 value, address manager);
    
    event RegistrationAccepted(uint256 indexed regId, uint256 indexed slotId);
    
    event RegistrationRejected(uint256 indexed regId);

    event Deposit(address indexed _sender, uint256 value);

    event AddRegistrant(address indexed _registrant);

    event SetRent(address indexed _account, uint16 indexed _period, uint256 indexed _amount);

    uint256 public regId;

    IOpsideSlots public opsideSlots;

    mapping(uint256 => Request) public requests;
    mapping(address => bool) public allowList;

    // mapping(address => uint256) public slotsManager;

    mapping(uint256 => uint256) public slots;
    mapping(uint256 => uint256) public regs;
    mapping(uint16 => uint256) public rents;

    modifier onlyEOA() {
        // Used to stop register from contracts (avoid accidentally lost tokens)
        require(!Address.isContract(msg.sender), "Account not EOA");
        _;
    }

    function initialize(address _opsideSlots) external virtual initializer {
        opsideSlots = IOpsideSlots(_opsideSlots);
        rents[182] = 500000 ether;
        rents[365] = 900000 ether;
        rents[730] = 1600000 ether;
        // Initialize OZ contracts
        __Ownable_init_unchained();
    }

    receive() external payable onlyEOA {
    //    revert("don't send to address");
        emit Deposit(msg.sender, msg.value);
    }

    /**
     * @notice Request to register a new rollup slot
     */
    function request(string calldata _name, address _manager, uint16 _period, uint256 _amount) external payable onlyEOA {
        require(_manager != address(0), "Invalid slot manager");
        require(!Address.isContract(_manager), "Manager not EOA");
        require(allowList[msg.sender], "Need to be allowed");
        address _token = opsideSlots.getIDEToken();
        
        if (_token == address(0)) {
            _amount = msg.value;
        } else {
            IERC20Upgradeable(_token).safeTransferFrom(msg.sender, address(this), _amount);
        }

        uint256 rent = rents[_period];
        require(rent > 0 && _amount >= rent, "Rent not enough");

        // require(slotsManager[_manager] == 0, "Manager already exists");
        
        regId++;
        // Store name 、manager、msg.value
        requests[regId] = Request(
            regId,
            _amount,
            _name,
            _manager
        );

        // Record manager
        // slotsManager[_manager] = regId;
        emit NewRegistration(regId, _amount, _manager);
    }

    /**
     * @notice Accept a request
     */
    function accept(uint256 _regId) external onlyOwner {
        require(_regId > 0 && _regId <= regId, "Invalid reg ID");
        require(requests[_regId].manager != address(0), "RegId does not exist when accept");
        require(regs[_regId] == 0, "registered");

        Request memory slot = requests[_regId];
        uint256 _amount = 0;
        uint256 _value = slot.value;
        if (opsideSlots.getIDEToken() != address(0)) {
            _amount = _value;
            _value = 0;
            IERC20Upgradeable(opsideSlots.getIDEToken()).approve(address(opsideSlots), _amount);
        }
        uint256 _slotId = opsideSlots.register{value: _value}(slot.name, slot.manager, _amount);
        // (bool success, bytes memory data) = address(opsideSlots).call{value: slot.value}(abi.encodeWithSelector(IOpsideSlots.register.selector, slot.name, slot.manager));
        
        // require(success, "register failed");
        
        // uint256 _slotId = abi.decode(data, (uint256));

        slots[_slotId] = _regId;
        regs[_regId] = _slotId;

        emit RegistrationAccepted(_regId, _slotId);
    }

    /**
     * @notice Reject a request
     */
    function reject(uint256 _regId) external onlyOwner {
        require(_regId > 0 && _regId <= regId, "Invalid reg ID");
        require(requests[_regId].manager != address(0), "regId does not exist when reject");

        // delete slotsManager[requests[_regId].manager];
        delete requests[_regId];

        emit RegistrationRejected(_regId);
    }

    /**
     * @notice Get details of a request
     */
    function getRequest(uint256 _regId) external view returns (Request memory req) {
        return requests[_regId];
    }


    /**
     * @notice Get regId by slotId
     */
    function getRegId(uint256 _slotId) external view returns (uint256) {
        return slots[_slotId];
    }

    /**
     * @notice Get total number of requests
     */
    function totalRequests() external view returns (uint256) {
        return regId;
    }

    function addRegistrant(address _registrant) external onlyOwner {
        require(_registrant != address(0), "Invalid slot manager");
        require(!Address.isContract(_registrant), "Manager not EOA");
        require(!allowList[_registrant], "Added");

        allowList[_registrant] = true;

        emit AddRegistrant(_registrant);
    }

    function setRent(uint16 _period, uint256 _amount) external onlyOwner {
        require(_amount >= 1 ether, "Need to be greater than 1");
        require(_period >= 182, "Minimum period 182");
        rents[_period] = _amount;

        emit SetRent(msg.sender, _period, _amount);
    }
}
