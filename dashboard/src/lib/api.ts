import { appConfig } from '$lib/config';

type RequestOptions = RequestInit & {
	token?: string;
	fetcher?: typeof fetch;
};

export async function apiFetch<T>(path: string, options: RequestOptions = {}): Promise<T> {
	const fetcher = options.fetcher || fetch;
	const headers = new Headers(options.headers);
	headers.set('Content-Type', 'application/json');

	const token = options.token || appConfig.adminApiToken;
	if (token) {
		headers.set('Authorization', `Bearer ${token}`);
	}

	const response = await fetcher(`${appConfig.apiBaseUrl}${path}`, {
		...options,
		headers
	});

	if (!response.ok) {
		throw new Error(`API request failed: ${response.status}`);
	}

	return response.json() as Promise<T>;
}
