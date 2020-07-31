import React, { useEffect, useState, useRef } from 'react';
import { Icon, Loader, } from '@pinpt/uic.next';
import {
	useIntegration,
	Account,
	AccountsTable,
	IntegrationType,
	OAuthConnect,
	Form,
	FormType,
	Http,
	IAPIKeyAuth,
	IOAuth2Auth,
	IAuth,
} from '@pinpt/agent.websdk';

import styles from './styles.module.less';
import { Verify } from 'crypto';
import { Item } from '@pinpt/uic.next/dist/SegmentedControl';

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
		return [res[1], res[0].map((item:any) => {
			item.id = ''+item.id;
			return item;
		})];
	}
	return [res[1], []]
}
async function fetchRepoCount(groupid: string, auth: IAPIKeyAuth | IOAuth2Auth): Promise<[number, number]> {
	// TODO: skip shared projects
	var url = auth.url + '/api/v4/groups/'+groupid+'/projects?with_shared=false';
	var res = await Http.get(url, { 'Authorization': createAuthHeader(auth) });
	// console.log("repos-count-res",JSON.stringify(res))
	if (res[1] === 200) {
		return [res[1], res[0].length];
	}
	return [res[1], 0]
}

const gitlabTeamToAccount = (data: any,count:number): Account => {
	return {
		id: data.id,
		name: data.name,
		description: data.description,
		avatarUrl: '',
		totalCount: 0,
		type: 'ORG',
		public: data.visibility == "private" ? false:true,
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

		console.log("config.accounts????????",JSON.stringify(config.accounts));
		
		const fetch = async () => {
			const data = await fetchWorkspaces(auth);
			const orgs = config.accounts || {};
			config.accounts = orgs;

			console.log("data",JSON.stringify(data))
			console.log("orgs",JSON.stringify(orgs))

			const newaccounts = data[1].map((org: any)=> gitlabTeamToAccount(org,0) );

			if (!installed) {
				newaccounts.forEach((account: Account) => (orgs[account.id] = account));
			}

			Object.keys(orgs).forEach((id: string) => {
				const found = newaccounts.find((acct: Account) => acct.id === orgs[id].id);

				if (!found) {
					const entry = orgs[id];
					newaccounts.push({...entry} as Account);
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
			description = 'For the selected accounts, all repositories, pull requests and other data will automatically be made available in Pinpoint once installed.'
			accounts = {accounts}
			entity = 'repo'
			config = {config}
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

const SelfManagedForm = ({ setAuth }: { setAuth: (val: IAuth) => void }) => {
	async function verify(auth: any): Promise<boolean> {
		setAuth(auth);
		return true;
	}
	return <Form type={FormType.API} name='GitLab' callback={verify} />;
};

const Integration = () => {
	const { loading,setLoading, currentURL, config, isFromRedirect, isFromReAuth, setConfig, authorization } = useIntegration();
	const [type, setType] = useState<IntegrationType | undefined>(config.integration_type);
	const [, setRerender] = useState(0);
	const [auth, setAuth] = useState<any>(undefined);
	const currentConfig = useRef(config);

	useEffect(() => {
		setRerender(Date.now());
	},[auth, config, setConfig]);

	useEffect(() => {
		console.log("useEffect1")
		if (!loading && authorization?.oauth2_auth) {
			config.integration_type = IntegrationType.CLOUD;
			config.oauth2_auth = {
				access_token: authorization.oauth2_auth.access_token,
				refresh_token: authorization.oauth2_auth.refresh_token,
				scopes: authorization.oauth2_auth.scopes,
			};

			setType(IntegrationType.CLOUD);
			setConfig(config);

			currentConfig.current = config;
		}
	}, [loading, authorization, config, setConfig]);

	// ============= OAuth 2.0 =============
	useEffect(() => {
		console.log("useEffect2")
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
						access_token: profile.Integration.auth.accessToken,
						refresh_token: profile.Integration.auth.refreshToken,
						scopes: profile.Integration.auth.scopes,
					};

					setType(IntegrationType.CLOUD)
					setConfig(config);

					currentConfig.current = config;
				}
			});
		}

	}, [loading, isFromRedirect, currentURL,config,setRerender,setConfig]);

	useEffect(() => {
		console.log("useEffect3")
		if (type) {
			console.log("useEffect6-0","auth",JSON.stringify(auth))
			config.integration_type = type;
			console.log("useEffect6-1","auth",JSON.stringify(auth))
			currentConfig.current =  config;

			console.log("useEffect6-2","auth",JSON.stringify(auth))
			setConfig(config);
			console.log("useEffect6-3","auth",JSON.stringify(auth))
			setRerender(Date.now());
			console.log("useEffect6-4","auth",JSON.stringify(auth))
		}
	}, [type, config, setConfig])

	if (loading) {
		return <Loader screen />;
	}

	let content;

	console.log("isFromReAuth",isFromReAuth)
	if (isFromReAuth) {
		if (config.integration_type === IntegrationType.CLOUD) {
			content = <OAuthConnect name='GitLab' reauth />;
		} else {
			content = <SelfManagedForm setAuth={setAuth}/>;
		}
	} else {
		if (!config.integration_type) {
			content = <LocationSelector setType={setType} />;
		} else if (config.integration_type === IntegrationType.CLOUD && !config.oauth2_auth) {
			content = <OAuthConnect name='GitLab' />;
		} else if (config.integration_type === IntegrationType.SELFMANAGED && !config.apikey_auth) {
			content = <SelfManagedForm setAuth={setAuth}/>;
		} else {
			content = <AccountList/>;
		}
	}

	return (
		<div className={styles.Wrapper}>
			{content}
		</div>
	);
};


export default Integration;