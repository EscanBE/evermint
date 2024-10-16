// SPDX-License-Identifier: MIT

pragma solidity >=0.7.0 <0.9.0;

struct DelegateMessage {
    string action;
    address delegator;
    string validator;
    uint256 amount;
    string denom;
    string oldValidator;
}

struct WithdrawRewardMessage {
    address delegator;
    string fromValidator;
}

interface IStakingCPC {
    /**
     * @dev Emitted when the delegator delegated into a validator.
     * `value` is delegation amount.
     */
    event Delegate(address indexed delegator, address indexed validator, uint256 value);

    /**
     * @dev Emitted when the delegator undelegated from a validator.
     * `value` is undelegation amount.
     */
    event Undelegate(address indexed delegator, address indexed validator, uint256 value);

    /**
     * @dev Emitted when the delegator withdraw reward from a validator.
     */
    event WithdrawReward(address indexed delegator, address indexed validator, uint256 value);

    /**
     * @dev Returns the name of the contract.
     */
    function name() external view returns (string memory);

    /**
     * @dev Returns the symbol of the staking denom.
     */
    function symbol() external view returns (string memory);

    /**
     * @dev Returns the decimals places of the staking denom.
     */
    function decimals() external view returns (uint8);

    /**
     * @dev Returns all the validators that the account delegated to.
     */
    function delegatedValidators(address account) external view returns (address[] memory);

    /**
     * @dev Returns the delegation of an account on a specific validator.
     */
    function delegationOf(address account, address validator) external view returns (uint256);

    /**
     * @dev Returns the total delegation of an account across all validators.
     */
    function totalDelegationOf(address account) external view returns (uint256);

    /**
     * @dev Returns the delegation reward of an account on a validator.
     */
    function rewardOf(address account, address validator) external view returns (uint256);

    /**
     * @dev Returns the delegation reward of an account across all validators.
     */
    function rewardsOf(address account) external view returns (uint256);

    /**
     * @dev Delegate a `value` amount of staking coin from the caller's account to `validator`.
     *
     * Returns a boolean value indicating whether the operation succeeded.
     *
     * Emits {Delegate} + {?WithdrawReward} events.
     */
    function delegate(address validator, uint256 value) external returns (bool);

    /**
     * @dev Delegate a `value` amount of staking coin from the caller's account to `validator`
     * using EIP-712.
     *
     * Returns a boolean value indicating whether the operation succeeded.
     *
     * Emits {Delegate} + {?WithdrawReward} events.
     */
    function delegateByMessage(DelegateMessage memory message, bytes32 r, bytes32 s, uint8 v) external returns (bool);

    /**
     * @dev Undelegate a `value` amount of staking coin of the caller's account from `validator`.
     *
     * Returns a boolean value indicating whether the operation succeeded.
     *
     * Emits {Undelegate} + {?WithdrawReward} events.
     */
    function undelegate(address validator, uint256 value) external returns (bool);

    /**
     * @dev Undelegate a `value` amount of staking coin of the caller's account from `validator`
     * using EIP-712.
     *
     * Returns a boolean value indicating whether the operation succeeded.
     *
     * Emits {Undelegate} + {?WithdrawReward} events.
     */
    function undelegateByMessage(DelegateMessage memory message, bytes32 r, bytes32 s, uint8 v) external returns (bool);

    /**
     * @dev Redelegate moves a `value` amount of staking coin of the caller's account from `srcValidator` to `dstValidator`.
     *
     * Returns a boolean value indicating whether the operation succeeded.
     *
     * Emits {Undelegate} + {Delegate} + {?WithdrawReward} event.
     */
    function redelegate(address srcValidator, address dstValidator, uint256 value) external returns (bool);

    /**
     * @dev Redelegate moves a `value` amount of staking coin of the caller's account from `srcValidator` to `dstValidator`
     * using EIP-712.
     *
     * Returns a boolean value indicating whether the operation succeeded.
     *
     * Emits {Undelegate} + {Delegate} + {?WithdrawReward} event.
     */
    function redelegateByMessage(DelegateMessage memory message, bytes32 r, bytes32 s, uint8 v) external returns (bool);

    /**
     * @dev Withdraw the caller's account staking reward from a single `validator`.
     *
     * Returns a boolean value indicating whether the operation succeeded.
     *
     * Emits a {WithdrawReward} event.
     */
    function withdrawReward(address validator) external returns (bool);

    /**
     * @dev Withdraw the caller's account staking reward from all delegated validators.
     *
     * Returns a boolean value indicating whether the operation succeeded.
     *
     * Emits multiple {WithdrawReward} events, one per validator.
     */
    function withdrawRewards() external returns (bool);

    /**
     * @dev Withdraw the caller's account staking reward from one or all delegated validators.
     *
     * Returns a boolean value indicating whether the operation succeeded.
     *
     * Emits multiple {WithdrawReward} events, one per validator.
     */
    function withdrawRewardsByMessage(WithdrawRewardMessage memory message, bytes32 r, bytes32 s, uint8 v) external returns (bool);

    // Trick with ERC-20 interface

    /**
     * This contract re-uses additional 2 methods of ERC-20 so staking can be simply as:
     * - Import to Metamask just like an ERC-20 token.
     * - Stake/Restake can be done via self-transfer method (read `transfer(address,uint256)` bellow for more information).
     */

    /**
     * @dev Returns the available balance plus delegation reward across all validators.
     */
    function balanceOf(address account) external view returns (uint256);

    /**
     * @dev Claims available staking reward and then delegate.
     * Rules:
     * - To avoid mistake (interact with wrong contract) that might cause fund lost, `to` must be self-address.
     * - `value` must be lower or equals to `available balance + unclaimed staking reward`.
     * - Validator to delegate to will be selected by the following rules:
     *   + If not delegated into any validator, a mid-power validator will be selected and receive delegation.
     *   + If delegated into one validator, that validator will receive delegation.
     *   + If delegated into many validators, the lowest power validator will receive delegation.
     */
    function transfer(address to, uint256 value) external returns (bool);
}