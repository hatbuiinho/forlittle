import { apiFetch } from '$lib/api';
import type { AuthResponse } from '$lib/types';

export async function getCurrentUser() {
	const response = await apiFetch<AuthResponse>('/api/v1/admin/auth/me');
	return response.user;
}

export async function login(email: string, password: string) {
	const response = await apiFetch<AuthResponse>('/api/v1/admin/auth/login', {
		method: 'POST',
		body: JSON.stringify({ email, password })
	});
	return response.user;
}

export async function logout() {
	await apiFetch<{ ok: boolean }>('/api/v1/admin/auth/logout', {
		method: 'POST'
	});
}
