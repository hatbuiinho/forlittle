import { appConfig } from '$lib/config';

type RequestOptions = RequestInit & {
	fetcher?: typeof fetch;
};

export class ApiAuthError extends Error {
	constructor() {
		super('Authentication required.');
		this.name = 'ApiAuthError';
	}
}

export async function apiFetch<T>(path: string, options: RequestOptions = {}): Promise<T> {
	const fetcher = options.fetcher || fetch;
	const headers = new Headers(options.headers);
	if (options.body && !headers.has('Content-Type')) {
		headers.set('Content-Type', 'application/json');
	}

	const response = await fetcher(`${appConfig.apiBaseUrl}${path}`, {
		...options,
		credentials: 'include',
		headers
	});

	if (response.status === 401) {
		if (typeof window !== 'undefined' && !window.location.pathname.startsWith('/login')) {
			window.location.href = `/login?next=${encodeURIComponent(window.location.pathname)}`;
		}

		throw new ApiAuthError();
	}

	if (!response.ok) {
		throw new Error(`API request failed: ${response.status}`);
	}

	if (response.status === 204) {
		return undefined as T;
	}

	return response.json() as Promise<T>;
}
