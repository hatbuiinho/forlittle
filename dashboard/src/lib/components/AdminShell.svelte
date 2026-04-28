<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { getCurrentUser, logout } from '$lib/auth';
	import { onMount } from 'svelte';
	import type { User } from '$lib/types';

	const navItems = [
		{
			href: '/',
			label: 'Overview',
			description: 'Machine monitoring scope and stack status',
			icon: 'icon-[solar--widget-4-bold-duotone]'
		},
		{
			href: '/machines',
			label: 'Machines',
			description: 'Track enrolled laptops and machine names',
			icon: 'icon-[solar--laptop-bold-duotone]'
		},
		{
			href: '/logs',
			label: 'Logs',
			description: 'Review browsing activity by machine',
			icon: 'icon-[solar--document-text-bold-duotone]'
		},
		{
			href: '/rules',
			label: 'Rules',
			description: 'Maintain allowlist and blocklist rules',
			icon: 'icon-[solar--shield-keyhole-bold-duotone]'
		}
	] as const;

	let { title, summary, children } = $props<{
		title: string;
		summary: string;
		children: () => unknown;
	}>();

	let isSidebarOpen = $state(false);
	let currentUser = $state<User | null>(null);
	let authReady = $state(false);

	function closeSidebar() {
		isSidebarOpen = false;
	}

	async function handleLogout() {
		await logout();
		await goto(resolve('/login'));
	}

	onMount(async () => {
		try {
			currentUser = await getCurrentUser();
			authReady = true;
		} catch {
			await goto(
				resolve(`/login?next=${encodeURIComponent(page.url.pathname)}` as `/login?${string}`)
			);
		}
	});
</script>

<div class="min-h-dvh">
	<button
		type="button"
		class={[
			'fixed inset-0 z-20 cursor-pointer bg-black/45 transition-opacity duration-200 lg:hidden',
			isSidebarOpen ? 'pointer-events-auto opacity-100' : 'pointer-events-none opacity-0'
		]}
		aria-label="Close navigation"
		onclick={closeSidebar}
	></button>

	<header
		class="sticky top-0 z-10 flex items-center justify-between gap-3 px-4 pt-4 pb-1 lg:hidden"
	>
		<button
			type="button"
			class="flex h-12 w-12 cursor-pointer flex-col items-center justify-center gap-1 rounded-full border border-black/10 bg-white/90 shadow-[0_20px_48px_rgba(0,0,0,0.08)]"
			aria-expanded={isSidebarOpen}
			aria-label="Toggle navigation"
			onclick={() => (isSidebarOpen = !isSidebarOpen)}
		>
			<span class="h-0.5 w-4 rounded-full bg-[#25441f]"></span>
			<span class="h-0.5 w-4 rounded-full bg-[#25441f]"></span>
			<span class="h-0.5 w-4 rounded-full bg-[#25441f]"></span>
		</button>
		<div class="min-w-0 flex-1">
			<p class="mb-1 text-xs font-bold tracking-[0.14em] text-[#555555] uppercase">For Little</p>
			<h1 class="m-0 text-base font-semibold text-[#111111]">Admin Dashboard</h1>
		</div>
		<div
			class="flex h-11 w-11 items-center justify-center rounded-full border border-black/10 bg-white/95 shadow-[0_20px_48px_rgba(0,0,0,0.08)]"
		>
			<span class="icon-[solar--shield-check-bold-duotone] text-xl text-[#25441f]"></span>
		</div>
	</header>

	<aside
		class={[
			'fixed top-0 bottom-0 left-0 z-30 grid w-[min(320px,calc(100vw-2rem))] grid-rows-[auto_auto_1fr_auto] gap-8 overflow-y-auto bg-[#0a0a0a] bg-[radial-gradient(circle_at_top_left,rgba(255,255,255,0.08),transparent_24%),linear-gradient(180deg,rgba(5,5,5,0.98),#0a0a0a)] px-6 py-8 text-[#f8f8f8] shadow-[0_28px_72px_rgba(5,12,8,0.34)] transition-transform duration-200 lg:fixed lg:h-dvh lg:w-80 lg:translate-x-0',
			isSidebarOpen ? 'translate-x-0' : '-translate-x-full'
		]}
	>
		<div class="flex items-start gap-4">
			<div
				class="flex h-12 w-12 items-center justify-center rounded-2xl border border-white/10 bg-white/10"
			>
				<span class="icon-[solar--shield-network-bold-duotone] text-2xl text-[#f3f3f3]"></span>
			</div>
			<div>
				<p class="mb-2 text-xs font-bold tracking-[0.14em] text-[#f0f0f0] uppercase">For Little</p>
				<h1 class="m-0 text-lg font-semibold">Admin Dashboard</h1>
				<p class="mt-2 leading-relaxed text-[#dbdbdb]/75">
					Machine-centric browsing governance for shared laptops.
				</p>
			</div>
		</div>

		<button
			type="button"
			class="w-fit cursor-pointer rounded-full border border-black/10 bg-white/90 px-4 py-2 text-[#111111] shadow-[0_20px_48px_rgba(0,0,0,0.08)] lg:hidden"
			aria-label="Close navigation"
			onclick={closeSidebar}
		>
			Close
		</button>

		<nav class="grid gap-3">
			{#each navItems as item (item.href)}
				<a
					class={[
						'grid grid-cols-[auto_1fr] items-start gap-3 rounded-2xl border p-4 transition duration-200 hover:translate-x-1',
						page.url.pathname === item.href
							? 'border-white/20 bg-white/10'
							: 'border-white/10 bg-white/[0.04] hover:border-white/20 hover:bg-white/10'
					]}
					href={resolve(item.href)}
					onclick={closeSidebar}
				>
					<span class={`${item.icon} text-xl text-[#efefef]`}></span>
					<div>
						<strong class="block font-semibold">{item.label}</strong>
						<small class="text-[#dbdbdb]/75">{item.description}</small>
					</div>
				</a>
			{/each}
		</nav>

		<div class="rounded-2xl border border-white/10 bg-white/[0.04] p-4">
			<p class="m-0 text-xs font-bold tracking-[0.14em] text-[#dbdbdb]/75 uppercase">Signed In</p>
			<p class="mt-2 mb-1 font-semibold">{currentUser?.display_name || 'Admin'}</p>
			<p class="m-0 text-sm break-words text-[#dbdbdb]/75">{currentUser?.email || '-'}</p>
			<button
				type="button"
				class="mt-4 w-full cursor-pointer rounded-xl border border-white/10 bg-white/90 px-4 py-2 font-bold text-[#111111]"
				onclick={handleLogout}
			>
				Logout
			</button>
		</div>
	</aside>

	<main class="p-4 lg:ml-80 lg:h-dvh lg:overflow-y-auto lg:p-8">
		<header class="mb-5 grid max-w-3xl gap-3 px-0.5 py-1">
			<div class="flex items-center gap-2">
				<span class="icon-[solar--pulse-2-bold-duotone] text-base text-[#25441f]"></span>
				<p class="m-0 text-xs font-bold tracking-[0.14em] text-[#3d3d3d] uppercase">Phase 1</p>
			</div>
			<div>
				<h2 class="m-0 text-3xl font-semibold text-[#111111] sm:text-4xl">{title}</h2>
				<p class="mt-2 max-w-2xl leading-relaxed text-[#555555]">{summary}</p>
			</div>
		</header>

		<section
			class="rounded-3xl border border-black/10 bg-white/90 p-4 shadow-[0_24px_60px_rgba(0,0,0,0.08)] backdrop-blur md:p-6"
		>
			{#if authReady}
				{@render children()}
			{:else}
				<p class="m-0 leading-relaxed text-[#555555]">Checking admin session...</p>
			{/if}
		</section>
	</main>
</div>
