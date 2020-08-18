import React, { useEffect, useState, useCallback, useRef, useMemo } from 'react';
import { Icon, Button, Loader, Theme } from '@pinpt/uic.next';
import {
	useIntegration,
	Account,
	AccountsTable,
	IntegrationType,
	OAuthConnect,
	ISession,
	Form,
	FormType,
	Config,
	Http,
	IAPIKeyAuth,
	IOAuth2Auth,
	IAuth,
	IInstalledLocation,
	ConfigAccount,
} from '@pinpt/agent.websdk';

import styles from './styles.module.less';
import { Verify } from 'crypto';
import { Item } from '@pinpt/uic.next/dist/SegmentedControl';

type Maybe<T> = T | undefined | null;

enum State {
	Location = 1,
	CloudSetup,
	SelfSetup,
	AgentSelector,
	Link,
	Validate,
	Repos,
}

interface workspacesResponse {
	id: string;
	name: string;
	description: string;
	visibility: string;
}

function createAuthHeader(auth: IAPIKeyAuth | IOAuth2Auth): string {
	if ('apikey' in auth) {
		var apiauth = (auth as IAPIKeyAuth);
		return 'bearer ' + apiauth.apikey;
	}
	var oauth = (auth as IOAuth2Auth);
	return 'bearer ' + oauth.access_token;
}
// TODO: add pagination for groups
async function fetchWorkspaces(auth: IAPIKeyAuth | IOAuth2Auth): Promise<[number, workspacesResponse[]]> {
	var url = auth.url + '/api/v4/groups?top_level_only=true';
	var res = await Http.get(url, { 'Authorization': createAuthHeader(auth) });
	// console.log("group-res",JSON.stringify(res))
	if (res[1] === 200) {
		return [res[1], res[0].map((item: any) => {
			item.id = '' + item.id;
			return item;
		})];
	}
	return [res[1], []]
}
async function fetchRepoCount(groupid: string, auth: IAPIKeyAuth | IOAuth2Auth): Promise<[number, number]> {
	// TODO: skip shared projects
	var url = auth.url + '/api/v4/groups/' + groupid + '/projects?with_shared=false';
	var res = await Http.get(url, { 'Authorization': createAuthHeader(auth) });
	// console.log("repos-count-res",JSON.stringify(res))
	if (res[1] === 200) {
		return [res[1], res[0].length];
	}
	return [res[1], 0]
}

const gitlabTeamToAccount = (data: any, count: number): Account => {
	return {
		id: data.id,
		name: data.name,
		description: data.description,
		avatarUrl: '',
		totalCount: 0,
		type: 'ORG',
		public: data.visibility == "private" ? false : true,
	};
};

const AccountList = () => {
	const { config, setLoading, setConfig, installed, setInstallEnabled } = useIntegration();
	const [accounts, setAccounts] = useState<Account[]>([]);

	let auth: IAPIKeyAuth | IOAuth2Auth;
	if (config.apikey_auth) {
		auth = config.apikey_auth as IAPIKeyAuth;
	} else {
		auth = config.oauth2_auth as IOAuth2Auth;
	}

	useEffect(() => {

		const fetch = async () => {
			const data = await fetchWorkspaces(auth);
			const orgs = config.accounts || {};
			config.accounts = orgs;

			console.log("data", JSON.stringify(data))
			console.log("orgs", JSON.stringify(orgs))

			const newaccounts = data[1].map((org: any) => gitlabTeamToAccount(org, 0));

			if (!installed) {
				newaccounts.forEach((account: Account) => (orgs[account.id] = account));
			}

			Object.keys(orgs).forEach((id: string) => {
				const found = newaccounts.find((acct: Account) => acct.id === orgs[id].id);

				if (!found) {
					const entry = orgs[id];
					newaccounts.push({ ...entry } as Account);
				}
			});

			for (var i = 0; i < newaccounts.length; i++) {
				var workspace = newaccounts[i];
				try {
					let res = await fetchRepoCount(workspace.id, auth);
					if (res[0] != 200) {
						console.error('error fetching repo count, response code', res[0]);
					}
					workspace.totalCount = res[1]
				} catch (ex) {
					console.error('error fetching repo count', ex);
				}
			}

			setAccounts(newaccounts);
			setInstallEnabled(installed ? true : Object.keys(config.accounts).length > 0);
			setConfig(config);

		}

		// TODO: Fix this setLoading doesn't work
		setLoading(true);
		fetch();
		setLoading(false);
	}, [installed, setInstallEnabled, config, setConfig, setLoading]);


	return (
		<AccountsTable
			description='For the selected accounts, all repositories, pull requests and other data will automatically be made available in Pinpoint once installed.'
			accounts={accounts}
			entity='repo'
			config={config}
		/>
	);
};

const LocationSelector = ({ setType }: { setType: (val: IntegrationType) => void }) => {
	return (
		<div className={styles.Location}>
			<div className={styles.Button} onClick={() => setType(IntegrationType.CLOUD)}>
				<Icon icon={['fas', 'cloud']} className={styles.Icon} />
				I'm using the <strong>GitLab.com</strong> cloud service to manage my data
			</div>

			<div className={styles.Button} onClick={() => setType(IntegrationType.SELFMANAGED)}>
				<Icon icon={['fas', 'server']} className={styles.Icon} />
				I'm using <strong>my own systems</strong> or a <strong>third-party</strong> to manage a GitLab service
			</div>
		</div>
	);
};

const AgentSelector = ({ setType }: { setType: (val: IntegrationType) => void }) => {
	const { selfManagedAgent, setSelfManagedAgentRequired } = useIntegration();
	const agentEnabled = selfManagedAgent?.enrollment_id;
	const agentRunning = selfManagedAgent?.running;
	const enabled = agentEnabled && agentRunning;
	return (
		<div className={styles.Location}>
			<div className={[styles.Button, enabled ? '' : styles.Disabled].join(' ')} onClick={() => enabled ? setType(IntegrationType.SELFMANAGED) : null}>
				<Icon icon={['fas', 'lock']} className={styles.Icon} />
				I'm using the <strong>Atlassian Jira Server</strong> behind a firewall which is not publically accessible
				<div>
					{agentEnabled && agentRunning ? (
						<>
							<Icon icon="info-circle" color={Theme.Mono300} />
							Your self-managed cloud agent will be used
						</>
					) : !agentEnabled ? (
						<>
							<div><Icon icon="exclamation-circle" color={Theme.Red500} /> You must first setup a self-managed cloud agent</div>
							<Button className={styles.Setup} color="Green" weight={500} onClick={(e: any) => {
								setSelfManagedAgentRequired();
								e.stopPropagation();
							}}>Setup</Button>
						</>
					) : (
								<>
									<div><Icon icon="exclamation-circle" color={Theme.Red500} /> Your agent is not running</div>
									<Button className={styles.Setup} color="Green" weight={500} onClick={(e: any) => {
										setSelfManagedAgentRequired();
										e.stopPropagation();
									}}>Configure</Button>
								</>
							)}
				</div>
			</div>

			<div className={styles.Button} onClick={() => setType(IntegrationType.CLOUD)}>
				<Icon icon={['fas', 'cloud']} className={styles.Icon} />
				I'm using the <strong>Atlassian Jira Server</strong> and it is publically accessible or whitelisted for Pinpoint
				<div>
					<Icon icon="check-circle" color={Theme.Mono300} /> Pinpoint will directly connect to your server
				</div>
			</div>
		</div>
	);
};

const SelfManagedForm = () => {
	async function verify(auth: any): Promise<void> {
		// setAuth(auth);
		// return true;
	}
	return <Form type={FormType.API} name='GitLab' callback={verify} />;
};

enum selfManagedFormState {
	EnteringUrl,
	Validating,
	Validated,
	Setup,
}

// const SelfManagedForm = ({session, callback, type}: {session: ISession, callback: (err: Error | undefined, url?: string) => void, type: IntegrationType}) => {
// 	const { setOAuth1Connect, setValidate, id } = useIntegration();
// 	const [buttonText, setButtonText] = useState('Validate');
// 	const url = useRef<string>();
// 	const timer = useRef<any>();
// 	const windowRef = useRef<any>();
// 	const state = useRef<selfManagedFormState>(selfManagedFormState.EnteringUrl);
// 	const [updatedState, setUpdatedState] = useState<selfManagedFormState>();
// 	const [, setRender] = useState(0);
// 	const ref = useRef<any>();
// 	const copy = useCallback(() => {
// 		if (ref.current) {
// 			ref.current.select();
// 			ref.current.setSelectionRange(0, 99999);
// 			document.execCommand('copy');
// 		}
// 	}, [ref]);
// 	useEffect(() => {
// 		return () => {
// 			setOAuth1Connect(''); // unset it
// 			if (timer.current) {
// 				clearInterval(timer.current);
// 				timer.current = null;
// 			}
// 			if (windowRef.current) {
// 				windowRef.current.close();
// 				windowRef.current = null;
// 			}
// 			ref.current = null;
// 			url.current = '';
// 		};
// 	}, [setOAuth1Connect]);
// 	useEffect(() => {
// 		if (updatedState) {
// 			state.current = updatedState;
// 			setRender(Date.now());
// 			if (updatedState === selfManagedFormState.Validated) {
// 				setTimeout(copy, 10);
// 			}
// 		}
// 	}, [updatedState, copy]);
// 	const verify = useCallback(async(auth: IAuth | string) => {
// 		switch (state.current) {
// 			case selfManagedFormState.EnteringUrl: {
// 				setButtonText('Cancel');
// 				state.current = selfManagedFormState.Validating;
// 				const config: Config = {
// 					integration_type: type,
// 					url: auth,
// 					action: 'VALIDATE_URL',
// 				};
// 				try {
// 					await setValidate(config);
// 					setButtonText('Begin Setup');
// 					setUpdatedState(selfManagedFormState.Validated);
// 				} catch (ex) {
// 					setButtonText('Validate');
// 					setUpdatedState(selfManagedFormState.EnteringUrl);
// 					callback(ex);
// 				}
// 				break;
// 			}
// 			case selfManagedFormState.Validating: {
// 				// if we get here, we clicked cancel so reset the state
// 				setButtonText('Validate');
// 				state.current = selfManagedFormState.EnteringUrl;
// 				callback(undefined);
// 				break;
// 			}
// 			case selfManagedFormState.Validated: {
// 				// if (windowRef.current) {
// 				// 	clearInterval(timer.current);
// 				// 	timer.current = null;
// 				// 	windowRef.current.close();
// 				// 	windowRef.current = null;
// 				// 	callback(undefined, url.current);
// 				// 	return;
// 				// }
// 				// const u = new URL(auth as string);
// 				// setOAuth1Connect(u.toString(), (err: Maybe<Error>) => {
// 				// 	setConnected(true);
// 				// });
// 				// const width = window.screen.width < 1000 ? window.screen.width : 1000;
// 				// const height = window.screen.height < 700 ? window.screen.height : 700;
// 				// u.pathname = '/plugins/servlet/applinks/listApplicationLinks';
// 				// windowRef.current = window.open(u.toString(), undefined, `toolbar=no,location=yes,status=no,menubar=no,scrollbars=yes,resizable=yes,width=${width},height=${height}`);
// 				// if (!windowRef.current) {
// 				// 	callback(new Error(`couldn't open the window to ${auth}`));
// 				// 	return;
// 				// }
// 				// timer.current = setInterval(() => {
// 				// 	if (windowRef.current?.closed) {
// 				// 		clearInterval(timer.current);
// 				// 		timer.current = null;
// 				// 		windowRef.current.close();
// 				// 		windowRef.current = null;
// 				// 		callback(undefined, auth as string);
// 				// 	}
// 				// }, 500);
// 				// url.current = auth as string;
// 				setUpdatedState(selfManagedFormState.Setup);
// 				setButtonText('Complete Setup');
// 				break;
// 			}
// 			case selfManagedFormState.Setup: {
// 				if (timer.current) {
// 					clearInterval(timer.current);
// 					timer.current = null;
// 				}
// 				if (windowRef.current) {
// 					windowRef.current.close();
// 					windowRef.current = null;
// 				}
// 				// setOAuth1Connect('');
// 				setTimeout(() => callback(undefined, url.current), 1);
// 				break;
// 			}
// 			default: break;
// 		}
// 	}, [callback, setOAuth1Connect, setValidate, type]);
// 	const seed = useMemo(() => String(Date.now()), []);
// 	let otherbuttons: React.ReactElement | undefined = undefined;
// 	if (state.current === selfManagedFormState.Setup) {
// 		otherbuttons = (
// 			<Button onClick={() => {
// 				// reset everything
// 				if (timer.current) {
// 					clearInterval(timer.current);
// 					timer.current = null;
// 				}
// 				if (windowRef.current) {
// 					windowRef.current.close();
// 					windowRef.current = null;
// 				}
// 				setButtonText('Validate');
// 				setUpdatedState(selfManagedFormState.EnteringUrl);
// 				url.current = undefined;
// 				setOAuth1Connect('');
// 			}}>Cancel</Button>
// 		);
// 	}
// 	return (
// 		<Form
// 			type={FormType.URL}
// 			name='Jira'
// 			title='Connect Pinpoint to Jira.'
// 			intro={<>Please provide the URL to your Jira instance and click the button to begin. A new window will open to your Jira instance to authorize Pinpoint to communicate with Jira. Once authorized, come back to this window to complete the connection process. <a rel="noopener noreferrer" target="_blank" href="https://www.notion.so/Pinpoint-Knowledge-Center-c624dd8935454394a3e91dd82bfe341c">Help</a></>}
// 			button={buttonText}
// 			callback={verify}
// 			readonly={state.current === selfManagedFormState.Setup}
// 			urlFormatter={formatUrl}
// 			afterword={() => {
// 				switch (state.current) {
// 					case selfManagedFormState.EnteringUrl: {
// 						return <></>;
// 					}
// 					case selfManagedFormState.Validating: {
// 						return (
// 							<div className={styles.Validating}>
// 								<Icon icon={['fas', 'spinner']} spin /> Validating
// 							</div>
// 						);
// 					}
// 					default: break;
// 				}
// 				const env = session.env === 'edge' ? 'edge.' : '';
// 				return (
// 					<div className={styles.Afterword}>
// 						<label htmlFor="instructions">Copy this URL and enter it in the "Create new link" field in Jira</label>
// 						<input ref={ref} type="text" name="instructions" onFocus={copy} readOnly value={`https://auth.api.${env}pinpoint.com/oauth1/jira/${id}/${seed.charAt(seed.length - 1)}`} />
// 					</div>
// 				);
// 			}}
// 			otherbuttons={otherbuttons}
// 			enabledValidator={async (url: IAuth | string) => {
// 				if (url && URLValidator(url as string)) {
// 					return true;
// 				}
// 				return false;
// 			}}
// 		/>
// 	);
// };

const makeAccountsFromConfig = (config: Config) => {
	return Object.keys(config.accounts ?? {}).map((key: string) => config.accounts?.[key]) as Account[];
};

const Integration = () => {
	const [state, setState] = useState<State>(State.Location);
	const { loading, installed, setInstallEnabled, currentURL, config, isFromRedirect, isFromReAuth, setConfig, authorization, setValidate } = useIntegration();
	const [type, setType] = useState<IntegrationType | undefined>(config.integration_type);
	const accounts = useRef<Account[]>([]);
	const [error, setError] = useState<Error | undefined>();
	const currentConfig = useRef<Config>(config);

	console.log(`{"epoch":1597717816191,"offset":-300,"rfc3339":"2020-08-17T21:30:16.191-05:00"}`);
	console.log("loading", loading);
	console.log("installed", installed);
	console.log("currentURL", currentURL);
	console.log("isFromRedirect", isFromRedirect);
	console.log("isFromReAuth", isFromReAuth);
	console.log("authorization", JSON.stringify(authorization));
	console.log("config", JSON.stringify(config));
	console.log("currentConfig", JSON.stringify(currentConfig));
	console.log("accounts", JSON.stringify(accounts));

	useEffect(() => {
		console.log("useEffect-config")
		console.log("\tconfig => ", JSON.stringify(config));
	}, [config]);

	useEffect(() => {
		console.log("useEffect-currentConfig")
		console.log("\tcurrentConfig => ", JSON.stringify(currentConfig));
	}, [currentConfig]);

	useEffect(() => {
		console.log("useEffect-authorization")
		console.log("\tauthorization => ", JSON.stringify(authorization));
	}, [authorization]);


	useEffect(() => {
		console.log("useEffect-isFromRedirect")
		console.log("\tloading", loading);
		console.log("\tcurrentURL", currentURL);
		console.log("\isFromRedirect", isFromRedirect);

		if (!loading && isFromRedirect && currentURL) {

			const search = currentURL.split('?');
			const tok = search[1].split('&');
			tok.forEach(async token => {
				const t = token.split('=');
				const k = t[0];
				const v = t[1];
				if (k === 'profile') {
					const profile = JSON.parse(atob(decodeURIComponent(v)));
					config.integration_type = IntegrationType.CLOUD;
					config.oauth2_auth = {
						url: "https://gitlab.com",
						access_token: profile.Integration.auth.accessToken,
						refresh_token: profile.Integration.auth.refreshToken,
						scopes: profile.Integration.auth.scopes,
						date_ts: new Date().valueOf(),
					};

					console.log("authObj", " - ", config.oauth2_auth);

					// setType(IntegrationType.CLOUD)
					setConfig(config);
					setState(State.Validate);

					currentConfig.current = config;

					// useEffect(() => {
					// 	console.log("check5")
					// 	if (type) {
					// 		config.integration_type = type;
					// 		currentConfig.current =  config;

					// 		setConfig(config);
					// 		setRerender(Date.now());
					// 	}
					// }, [type, config, setConfig])

				}
			});
		}

	}, [loading, currentURL, isFromRedirect, setConfig])

	// useEffect(() => {
	// 	console.log("check1")
	// 	setRerender(Date.now());
	// },[auth, config, setConfig]);

	useEffect(() => {
		console.log("useEffect-installed");
		console.log("\tconfig => ",JSON.stringify(config));
		console.log("\installed => ",installed);
		if ((installed && accounts.current?.length === 0) || config?.accounts) {
			currentConfig.current = config;
			accounts.current = makeAccountsFromConfig(config);
			setState(State.Repos);
		} else if (currentConfig.current?.accounts) {
			accounts.current = makeAccountsFromConfig(currentConfig.current);
			setState(State.Repos);
		}

	}, [installed, config]);
	// }, [installed, config]);

	// useEffect(() => {
	// 	console.log("check2")
	// 	// const inupgrade = window.sessionStorage.getItem(upgradeStorageKey) === 'true';
	// 	// if (debugState) {
	// 	// 		console.log('JIRA: state machine', JSON.stringify({
	// 	// 		installed,
	// 	// 		inupgrade,
	// 	// 		upgradeRequired,
	// 	// 		accounts: config?.accounts,
	// 	// 		isFromReAuth,
	// 	// 		isFromRedirect,
	// 	// 		currentURL,
	// 	// 		insideRedirect: insideRedirect.current,
	// 	// 	}, null, 2));
	// 	// }
	// 	// if (upgradeRequired && !inupgrade) {
	// 	// 	setState(State.UpgradeRequired);
	// 	// } else if (inupgrade && !isFromRedirect) {
	// 	// 	setState(State.AgentSelector);
	// 	// } else 
	// 	// if (isFromReAuth) {
	// 	// 	setState(State.AgentSelector);
	// 	// } 
	// 	if (installed || config?.accounts) {
	// 		setState(State.Repos);
	// 	// 	if (installed && inupgrade) {
	// 	// 		completeUpgrade();
	// 		}
	// 	// } else if (isFromRedirect && currentURL && !insideRedirect.current) {
	// 	// 	const url = window.sessionStorage.getItem(urlStorageKey);
	// 	// 	if (url) {
	// 	// 		const search = currentURL.split('?');
	// 	// 		const tok = search[1].split('&');
	// 	// 		tok.some(token => {
	// 	// 			const t = token.split('=');
	// 	// 			const k = t[0];
	// 	// 			const v = t[1];
	// 	// 			if (k === 'result') {
	// 	// 				const result = JSON.parse(atob(decodeURIComponent(v)));
	// 	// 				const { success, consumer_key, oauth_token, oauth_token_secret, error } = result;
	// 	// 				if (success) {
	// 	// 					if (url) {
	// 	// 						const _config = { ...config };
	// 	// 						_config.oauth1_auth = {
	// 	// 							date_ts: Date.now(),
	// 	// 							url,
	// 	// 							consumer_key,
	// 	// 							oauth_token,
	// 	// 							oauth_token_secret,
	// 	// 						}
	// 	// 						currentConfig.current = _config;
	// 	// 						insideRedirect.current = true;
	// 	// 						setState(State.Validate);
	// 	// 					}
	// 	// 				} else {
	// 	// 					setError(new Error(error ?? 'Unknown error obtaining OAuth token'));
	// 	// 				}
	// 	// 				return true;
	// 	// 			}
	// 	// 			return false;
	// 	// 		});
	// 	// 	}
	// 	// } 
	// 	else if (accounts.current?.length > 0) {
	// 		setState(State.Repos);
	// 	}
	// }, [config,installed,  isFromReAuth, currentURL, isFromRedirect]);



	// useEffect(() => {
	// 	console.log("check3")
	// 	if (!loading && authorization?.oauth2_auth) {
	// 		config.integration_type = IntegrationType.CLOUD;
	// 		config.oauth2_auth = {
	// 			url : "https://gitlab.com",
	// 			access_token: authorization.oauth2_auth.access_token,
	// 			refresh_token: authorization.oauth2_auth.refresh_token,
	// 			scopes: authorization.oauth2_auth.scopes,
	// 			date_ts: new Date().valueOf(),
	// 		};

	// 		setType(IntegrationType.CLOUD);
	// 		setConfig(config);

	// 		currentConfig.current = config;
	// 	}
	// }, [loading, authorization, config, setConfig]);

	// // ============= OAuth 2.0 =============
	// useEffect(() => {
	// 	console.log("check4")
	// 	if (!loading && isFromRedirect && currentURL) {
	// 		const search = currentURL.split('?');
	// 		const tok = search[1].split('&');
	// 		tok.forEach(async token => {
	// 			const t = token.split('=');
	// 			const k = t[0];
	// 			const v = t[1];
	// 			if (k === 'profile') {
	// 				const profile = JSON.parse(atob(decodeURIComponent(v)));
	// 				config.integration_type = IntegrationType.CLOUD;
	// 				config.oauth2_auth = {
	// 					url : "https://gitlab.com",
	// 					access_token: profile.Integration.auth.accessToken,
	// 					refresh_token: profile.Integration.auth.refreshToken,
	// 					scopes: profile.Integration.auth.scopes,
	// 					date_ts: new Date().valueOf(),
	// 				};

	// 				setType(IntegrationType.CLOUD)
	// 				setConfig(config);
	// 				setState(State.Validate);

	// 				currentConfig.current = config;
	// 			}
	// 		});
	// 	}

	// }, [loading, isFromRedirect, currentURL,config,setRerender,setConfig]);

	// useEffect(() => {
	// 	console.log("check5")
	// 	if (type) {
	// 		config.integration_type = type;
	// 		currentConfig.current =  config;

	// 		setConfig(config);
	// 		setRerender(Date.now());
	// 	}
	// }, [type, config, setConfig])

	useEffect(() => {
		console.log("useEffect-state")
		console.log("\tstate", JSON.stringify(state))
		if (state === State.Validate && accounts.current?.length === 0) {
			const run = async () => {
				const _config = { ...currentConfig.current, action: 'FETCH_ACCOUNTS' };
				try {
					console.log("beforeSetValidate")
					const res =  await setValidate(_config);
					console.log("afterSetValidate")
					console.log("res", JSON.stringify(res))
					const newconfig = { ...currentConfig.current };
					newconfig.accounts = {};
					if (res?.accounts) {
						// res.accounts.ForEach( (item) => {
						// 	newconfig.accounts[item.id] = item;
						// });
						// newconfig.accounts[res.accounts.id] = res.accounts;
						var t = res.accounts as Account[];
						t.forEach(( item ) => {
							console.log("item",JSON.stringify(item))
							if ( newconfig  && newconfig.accounts){
								newconfig.accounts[item.id] = item;
							}
						});
					}
					currentConfig.current = newconfig;
					accounts.current = res.accounts as Account[];
					setInstallEnabled(Object.keys(newconfig.accounts).length > 0);
					setState(State.Repos);
					setConfig(currentConfig.current);
				} catch (err) {
					console.error(err);
					setError(err);
				}
			};
			run();
		}
	}, [setValidate, state, setConfig]);

	if (loading) {
		return <Loader screen />;
	}

	let content;

	if (isFromReAuth) {
		if (config.integration_type === IntegrationType.CLOUD) {
			content = <OAuthConnect name='GitLab' reauth />;
		} else {
			content = <SelfManagedForm />;
		}
	} else {
		switch (state) {
			case State.Location: {
				// console.log("check8")
				content = <LocationSelector setType={async (intType: IntegrationType) => {
					try {
						setType(intType);
						if (intType === IntegrationType.CLOUD) {
							setState(State.CloudSetup);
						} else {
							setState(State.SelfSetup);
						}
					} catch (err) {
						setError(err);
					}
				}} />;
				break;
			}
			case State.CloudSetup: {
				// console.log("check9")
				content = <OAuthConnect name='GitLab' />;
				break;
			}
			case State.SelfSetup: {
				// console.log("check10")
				content = <SelfManagedForm />;
				break;
			}
			case State.Validate: {
				// console.log("check11")
				content = (
					<Loader screen className={styles.Validate}>
						<div>
							<p>
								<Icon icon="check-circle" color={Theme.Green500} /> Connected
							</p>
							<p>Fetching Gitlab details...</p>
						</div>
					</Loader>
				);
				break;
			}
			case State.Repos: {
				// console.log("check12")
				content = (
					<AccountsTable
						description='For the selected accounts, all projects, issues and other data will automatically be made available in Pinpoint once installed.'
						accounts={accounts.current}
						entity='project'
						config={currentConfig.current}
					/>
				);
				break;
			}
		}
	}



	// console.log("isFromReAuth",isFromReAuth)
	// if (isFromReAuth) {
	// 	if (config.integration_type === IntegrationType.CLOUD) {
	// 		content = <OAuthConnect name='GitLab' reauth />;
	// 	} else {
	// 		content = <SelfManagedForm setAuth={setAuth}/>;
	// 	}
	// } else {
	// 	if (!config.integration_type) {
	// 		content = <LocationSelector setType={setType} />;
	// 	} else if (config.integration_type === IntegrationType.CLOUD && !config.oauth2_auth) {
	// 		content = <OAuthConnect name='GitLab' />;
	// 	} else if (config.integration_type === IntegrationType.SELFMANAGED && !config.apikey_auth) {
	// 		content = <SelfManagedForm setAuth={setAuth}/>;
	// 	} else {
	// 		content = <AccountList/>;
	// 	}
	// }

	return (
		<div className={styles.Wrapper}>
			{content}
		</div>
	);
};


export default Integration;