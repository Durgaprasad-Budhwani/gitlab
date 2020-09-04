import React, { useEffect, useState, useCallback,  useRef } from 'react';
import { Icon, Button, Loader, Theme } from '@pinpt/uic.next';
import {
	useIntegration,
	Account,
	AccountsTable,
	IntegrationType,
	OAuthConnect,
	Form,
	FormType,
	Config,
	IAPIKeyAuth,
} from '@pinpt/agent.websdk';

import styles from './styles.module.less';

type Maybe<T> = T | undefined | null;

enum State {
	IntegrationType = 1,
	Location,
	CloudSetup,
	SelfSetup,
	AgentSelector,
	Link,
	Validate,
	Repos,
}

const IntegrationTypeSelector = ({ isSourceCodeChecked, setIsSourceCodeChecked, isWorkChecked, setIsWorkChecked, setState }: { isSourceCodeChecked: boolean, setIsSourceCodeChecked: (val: boolean) => void, isWorkChecked: boolean, setIsWorkChecked: (val: boolean) => void, state: State,setState: (state: State)=> void}) => {
	return (
		<div>
			<div className="checkbox">
				<label>
					<input type="checkbox" value="SourceCode" checked={isSourceCodeChecked} onChange={()=>{
						setIsSourceCodeChecked(!isSourceCodeChecked)}
					} />SourceCode
				</label>
			</div>
			<div className="checkbox">
				<label>
					<input type="checkbox" value="Work" checked={isWorkChecked} onChange={()=>{
						setIsWorkChecked(!isWorkChecked)}
					} />Work
				</label>
			</div>
			<div>
			<Button className={styles.Setup} color="Green" weight={500} onClick={(e: any) => {
				// TODO: add validation to at least select either SouceCode or Work
				// TODO: persist selection into DB
				setState(State.Location)
				e.stopPropagation();
			}}>Next</Button>
			</div>
		</div>
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

const SelfManagedForm = ({callback}: {callback: (auth : IAPIKeyAuth) => void}) => {
	async function verify(auth: IAPIKeyAuth): Promise<void> {
		callback(auth);
	}
	return <Form type={FormType.API} name='GitLab' callback={verify} />;
};

const makeAccountsFromConfig = (config: Config) => {
	return Object.keys(config.accounts ?? {}).map((key: string) => config.accounts?.[key]) as Account[];
};

const Integration = () => {
	const [state, setState] = useState<State>(State.IntegrationType);
	const [isSourceCodeChecked, setIsSourceCodeChecked ] = useState<boolean>(false);
	const [isWorkChecked, setIsWorkChecked ] = useState<boolean>(false);
	const { loading, installed, setInstallEnabled, currentURL, config, isFromRedirect, isFromReAuth, setConfig, authorization, setValidate } = useIntegration();
	const [type, setType] = useState<IntegrationType | undefined>(config.integration_type);
	const accounts = useRef<Account[]>([]);
	const [error, setError] = useState<Error | undefined>();
	const currentConfig = useRef<Config>(config);

	useEffect(() => {
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
						date_ts: Date.now(),
					};

					setConfig(config);
					setState(State.Validate);

					currentConfig.current = config;

				}
			});
		}

	}, [loading, currentURL, isFromRedirect, setConfig]);

	useEffect(() => {
		if ((installed && accounts.current?.length === 0) || config?.accounts) {
			currentConfig.current = config;
			accounts.current = makeAccountsFromConfig(config);
			setState(State.Repos);
		} else if (currentConfig.current?.accounts) {
			accounts.current = makeAccountsFromConfig(currentConfig.current);
			setState(State.Repos);
		}

	}, [installed, config]);

	useEffect(() => {
		if (state === State.Validate && accounts.current?.length === 0) {
			const run = async () => {
				const _config = { ...currentConfig.current, action: 'FETCH_ACCOUNTS' };
				try {
					const res =  await setValidate(_config);
					const newconfig = { ...currentConfig.current };
					newconfig.accounts = {};
					if (res?.accounts) {
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

	const selfManagedCallback = useCallback((auth : IAPIKeyAuth) => {

		config.integration_type = IntegrationType.SELFMANAGED;
		config.apikey_auth = auth;

		setConfig(config);

		currentConfig.current = config;

		setState(State.Validate);

	}, [setState, config, setConfig]);

	if (loading) {
		return <Loader screen />;
	}

	let content;

	if (isFromReAuth) {
		if (config.integration_type === IntegrationType.CLOUD) {
			content = <OAuthConnect name='GitLab' reauth />;
		} else {
			content = <SelfManagedForm callback={selfManagedCallback} />;
		}
	} else {
		switch (state) {
			case State.IntegrationType: {
				content = <IntegrationTypeSelector 
					isSourceCodeChecked={isSourceCodeChecked}  
					setIsSourceCodeChecked={setIsSourceCodeChecked} 
					isWorkChecked={isWorkChecked} 
					setIsWorkChecked={setIsWorkChecked}
					state={state}
					setState={setState}/>
				break
			}
			case State.Location: {
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
				content = <OAuthConnect name='GitLab' />;
				break;
			}
			case State.SelfSetup: {
				content = <SelfManagedForm callback={selfManagedCallback} />;
				break;
			}
			case State.Validate: {
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

	return (
		<div className={styles.Wrapper}>
			{content}
		</div>
	);
};


export default Integration;