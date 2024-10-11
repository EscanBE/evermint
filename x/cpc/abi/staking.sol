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
     * Emits each time an account withdrawal reward, regardless number of validators.
     */
    event WithdrawReward(address indexed delegator);

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
     * Emits a {Delegate} event.
     */
    function delegate(address validator, uint256 value) external returns (bool);

    /**
     * @dev Undelegate a `value` amount of staking coin of the caller's account from `validator`.
     *
     * Returns a boolean value indicating whether the operation succeeded.
     *
     * Emits an {Undelegate} event.
     */
    function undelegate(address validator, uint256 value) external returns (bool);

    /**
     * @dev Redelegate moves a `value` amount of staking coin of the caller's account from `srcValidator` to `dstValidator`.
     *
     * Returns a boolean value indicating whether the operation succeeded.
     *
     * Emits an {Undelegate} event and then a {Delegate} event.
     */
    function redelegate(address srcValidator, address dstValidator, uint256 value) external returns (bool);

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
     * Emits a (single) {WithdrawReward} event.
     */
    function withdrawRewards() external returns (bool);
}