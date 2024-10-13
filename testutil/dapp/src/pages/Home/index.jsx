import preactLogo from '../../assets/preact.svg';
import { useState } from 'preact/hooks';
import { StakingContractAddress } from '../../constants/contracts';
import { StakingAbi } from '../../constants/abi';
import { ValidatorAddress, ValidatorCosmosAddress } from '../../constants/env';
import './style.css';
import { useSDK } from "@metamask/sdk-react";
import { ethers } from "ethers";

export function Home() {
	const [account, setAccount] = useState("");
	const { sdk, connected, connecting, provider, chainId } = useSDK();
	const [stakingPcResult, setStakingPcResult] = useState("");
	const [loading, setLoading] = useState(false);

	const connect = async () => {
		try {
			const accounts = await sdk?.connect();
			setAccount(accounts?.[0]);
		} catch (err) {
			console.warn("failed to connect..", err);
		}
	};
	const isFullyConnected = () => connected && chainId && account;

	const web3Provider = new ethers.BrowserProvider(window.ethereum);
	const getSigner = async () => {
		return await web3Provider.getSigner(account);
	};
	const execContract = async (address, abi, executor) => {
		try {
			setLoading(true);
			const signer = await getSigner();
			const contract = new ethers.Contract(
				address,
				abi,
				signer
			);
			return await executor(contract, signer);
		} finally {
			setLoading(false);
		}
	};
	const execStakingContract = async (executor) => {
		return await execContract(StakingContractAddress, StakingAbi, executor);
	};
	const execStakingContractAndPrint = async (executor) => {
		setStakingPcResult(`${await execStakingContract(executor)}`);
	};
	const toQueryGetReceipt = async (tx) => {
		return `curl http://localhost:8545 \
		-X POST \
		-H "Content-Type: application/json" \
		--data '{"method":"eth_getTransactionReceipt","params":["${tx.hash}"],"id":1,"jsonrpc":"2.0"}' | jq`;
	};
	const tryFetchReceiptAndPrint = async (tx) => {
		if (!tx) {
			return;
		}
		setLoading(true);
		let cancelled = false;
		try {
			setTimeout(() => {
				cancelled = true;
			}, 15_000);

			do {
				const receipt = await web3Provider.getTransactionReceipt(tx.hash);
				if (receipt && receipt.blockNumber) {
					setStakingPcResult(JSON.stringify(receipt, null, 2).replace(/\\n/g, ' \n '));
					cancelled = true;
				}
				await new Promise(r => setTimeout(r, 1_000));
			} while (!cancelled);
		} finally {
			setLoading(false);
		}
	};

	return (
		<div class="home">
			<a href="https://preactjs.com" target="_blank">
				<img src={preactLogo} alt="Preact logo" height="160" width="160" />
			</a>
			<h1>Simple DApp to interact with the blockchain</h1>
			{!isFullyConnected() && (
				<button style={{ padding: 10, margin: 10 }} onClick={connect}>
					Connect to MetaMask
				</button>
			)}
			{connected && (
				<div>
					<>
						{chainId && `Connected chain: ${chainId}`}
						<p></p>
						{account && `Connected account: ${account}`}
						<p></p>
						{`Hardcoded validator address: ${ValidatorAddress}`}
					</>
				</div>
			)}
			{isFullyConnected() && (
				<div id="precompiled-contracts">
					<hr />
					<h3>Precompiled contracts</h3>
					<div id="staking-cpc" className="pcSection">
						<h2>Staking</h2>
						<div>
							<textarea className="pcResult display-linebreak" cols={500} rows={10}>{stakingPcResult}</textarea>
						</div>
						<button disabled={loading} onClick={async () => {
							await execStakingContractAndPrint(async (contract, signer) => {
								return await contract.name();
							});
						}}>name()</button><br />
						<button disabled={loading} onClick={async () => {
							await execStakingContractAndPrint(async (contract, signer) => {
								return await contract.symbol();
							});
						}}>symbol()</button><br />
						<button disabled={loading} onClick={async () => {
							await execStakingContractAndPrint(async (contract, signer) => {
								return await contract.decimals();
							});
						}}>decimals()</button><br />
						<button disabled={loading} onClick={async () => {
							await execStakingContractAndPrint(async (contract, signer) => {
								return await contract.delegatedValidators(await signer.getAddress());
							});
						}}>delegatedValidators(address)</button><br />
						<button disabled={loading} onClick={async () => {
							await execStakingContractAndPrint(async (contract, signer) => {
								const symbol = await contract.symbol();
								const decimals = await contract.decimals();
								const delegation = await contract.delegationOf(await signer.getAddress(), ValidatorAddress);
								return `${ethers.formatUnits(delegation, decimals)} ${symbol}`;
							});
						}}>delegationOf(address,{ValidatorAddress}::address)</button><br />
						<button disabled={loading} onClick={async () => {
							await execStakingContractAndPrint(async (contract, signer) => {
								return await contract.totalDelegationOf(await signer.getAddress());
							});
						}}>totalDelegationOf(address)</button><br />
						<button disabled={loading} onClick={async () => {
							await execStakingContractAndPrint(async (contract, signer) => {
								const symbol = await contract.symbol();
								const decimals = await contract.decimals();
								const reward = await contract.rewardOf(await signer.getAddress(), ValidatorAddress);
								return `${ethers.formatUnits(reward, decimals)} ${symbol}`;
							});
						}}>rewardOf(address,{ValidatorAddress}::address)</button><br />
						<button disabled={loading} onClick={async () => {
							await execStakingContractAndPrint(async (contract, signer) => {
								const symbol = await contract.symbol();
								const decimals = await contract.decimals();
								const rewards = await contract.rewardsOf(await signer.getAddress());
								return `${ethers.formatUnits(rewards, decimals)} ${symbol}`;
							});
						}}>rewardsOf(address)</button><br />
						<button disabled={loading} onClick={async () => {
							await execStakingContractAndPrint(async (contract, signer) => {
								const retTx = await contract.delegate(ValidatorAddress, '1000000000000000000');
								tryFetchReceiptAndPrint(retTx);
								return toQueryGetReceipt(retTx);
							});
						}}>delegate({ValidatorAddress}::address,1^18::uint256)</button><br />
						<button disabled={loading} onClick={async () => {
							const signed = await window.ethereum.request({
								"method": "eth_signTypedData_v4",
								"params": [
									account,
									{
										types: {
											EIP712Domain: [
												{
													name: "name",
													type: "string"
												},
												{
													name: "version",
													type: "string"
												},
												{
													name: "chainId",
													type: "uint256"
												},
												{
													name: "verifyingContract",
													type: "address"
												}
											],
											Staking: [
												{
													name: "action",
													type: "string"
												},
												{
													name: "account",
													type: "address"
												},
												{
													name: "toValidator",
													type: "string"
												},
												{
													name: "fromValidator",
													type: "string"
												},
												{
													name: "amount",
													type: "uint256"
												},
												{
													name: "denom",
													type: "string"
												}
											]
										},
										primaryType: "Staking",
										domain: {
											name: "Staking - Precompiled Contract",
											version: "1",
											chainId: chainId,
											verifyingContract: StakingContractAddress
										},
										message: {
											action: "Delegate",
											account: account,
											toValidator: ValidatorCosmosAddress,
											fromValidator: "",
											amount: '1000000000000000000',
											denom: 'wei'
										}
									}
								],
							});
							const signedHex = `${signed}`;
							const signature = signedHex.substring(2);
							const r = "0x" + signature.substring(0, 64);
							const s = "0x" + signature.substring(64, 128);
							const v = parseInt(signature.substring(128, 130), 16);
							setStakingPcResult(` r = ${r} \n s = ${s} \n v = ${v}`);
						}}>delegateTyped (demo)</button><br />
						<button disabled={loading} onClick={async () => {
							await execStakingContractAndPrint(async (contract, signer) => {
								const retTx = await contract.undelegate(ValidatorAddress, '1000000000000000000');
								tryFetchReceiptAndPrint(retTx);
								return toQueryGetReceipt(retTx);
							});
						}}>undelegate({ValidatorAddress}::address,1^18::uint256)</button><br />
						<button disabled={true} onClick={async () => {
							await execStakingContractAndPrint(async (contract, signer) => {
								const retTx = await contract.redelegate(ValidatorAddress, ValidatorAddress, '1000000000000000000');
								tryFetchReceiptAndPrint(retTx);
								return toQueryGetReceipt(retTx);
							});
						}}>redelegate({ValidatorAddress}::address,address,1^18::uint256)</button><br />
						<button disabled={loading} onClick={async () => {
							await execStakingContractAndPrint(async (contract, signer) => {
								const retTx = await contract.withdrawReward(ValidatorAddress);
								tryFetchReceiptAndPrint(retTx);
								return toQueryGetReceipt(retTx);
							});
						}}>withdrawReward({ValidatorAddress}::address)</button><br />
						<button disabled={loading} onClick={async () => {
							await execStakingContractAndPrint(async (contract, signer) => {
								const retTx = await contract.withdrawRewards();
								tryFetchReceiptAndPrint(retTx);
								return toQueryGetReceipt(retTx);
							});
						}}>withdrawRewards()</button><br />
					</div>
				</div>
			)}
		</div>
	);
}
