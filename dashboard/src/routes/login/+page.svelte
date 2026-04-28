<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { login } from '$lib/auth';

	let email = $state('');
	let password = $state('');
	let loading = $state(false);
	let errorMessage = $state('');

	const allowedNextPaths = ['/', '/machines', '/logs', '/rules'] as const;

	function safeNextPath(value: string | null) {
		return allowedNextPaths.find((path) => path === value) || '/';
	}

	async function submitLogin() {
		if (!email.trim() || !password) {
			errorMessage = 'Email and password are required.';
			return;
		}

		loading = true;
		errorMessage = '';

		try {
			await login(email.trim(), password);
			await goto(resolve(safeNextPath(page.url.searchParams.get('next'))));
		} catch {
			errorMessage = 'Invalid email or password.';
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>Login | For Little</title>
</svelte:head>

<main
	class="grid min-h-dvh place-items-center bg-[#f5f1e8] bg-[radial-gradient(circle_at_top_left,rgba(37,68,31,0.14),transparent_32%),linear-gradient(135deg,#f7f3ea,#ebe2d2)] p-4"
>
	<section
		class="w-full max-w-md rounded-3xl border border-black/10 bg-white/95 p-6 shadow-[0_28px_72px_rgba(0,0,0,0.12)]"
	>
		<div class="mb-8">
			<p class="mb-2 text-xs font-bold tracking-[0.16em] text-[#25441f] uppercase">For Little</p>
			<h1 class="m-0 text-3xl font-semibold text-[#111111]">Admin Login</h1>
			<p class="mt-2 leading-relaxed text-[#555555]">
				Sign in to manage machines, visit logs, and browsing policy rules.
			</p>
		</div>

		<form class="grid gap-4" onsubmit={(event) => event.preventDefault()}>
			<label class="grid gap-1.5">
				<span class="text-xs font-bold tracking-[0.08em] text-[#555555] uppercase">Email</span>
				<input
					class="min-h-11 rounded-xl border border-black/10 bg-white px-3 text-[#111111] focus:border-[#25441f] focus:ring-[#25441f]"
					bind:value={email}
					type="email"
					autocomplete="username"
					placeholder="admin@example.com"
				/>
			</label>

			<label class="grid gap-1.5">
				<span class="text-xs font-bold tracking-[0.08em] text-[#555555] uppercase">Password</span>
				<input
					class="min-h-11 rounded-xl border border-black/10 bg-white px-3 text-[#111111] focus:border-[#25441f] focus:ring-[#25441f]"
					bind:value={password}
					type="password"
					autocomplete="current-password"
					placeholder="Enter password"
				/>
			</label>

			{#if errorMessage}
				<p class="m-0 rounded-xl bg-[#5f1818]/10 px-3 py-2 text-sm font-semibold text-[#5f1818]">
					{errorMessage}
				</p>
			{/if}

			<button
				type="button"
				class="min-h-11 cursor-pointer rounded-xl border border-[#25441f] bg-[#25441f] px-4 py-2 font-bold text-white disabled:cursor-not-allowed disabled:opacity-60"
				disabled={loading}
				onclick={submitLogin}
			>
				{loading ? 'Signing in...' : 'Sign In'}
			</button>
		</form>
	</section>
</main>
