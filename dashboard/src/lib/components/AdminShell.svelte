<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';

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

	function closeSidebar() {
		isSidebarOpen = false;
	}
</script>

<div class="shell">
	<button
		type="button"
		class:open={isSidebarOpen}
		class="sidebar-backdrop"
		aria-label="Close navigation"
		onclick={closeSidebar}
	></button>

	<header class="mobile-topbar">
		<button
			type="button"
			class="menu-button"
			aria-expanded={isSidebarOpen}
			aria-label="Toggle navigation"
			onclick={() => (isSidebarOpen = !isSidebarOpen)}
		>
			<span></span>
			<span></span>
			<span></span>
		</button>
		<div class="mobile-topbar__copy">
			<p class="eyebrow">For Little</p>
			<h1>Admin Dashboard</h1>
		</div>
		<div class="mobile-topbar__status">
			<span class="icon-[solar--shield-check-bold-duotone]"></span>
		</div>
	</header>

	<aside class:open={isSidebarOpen} class="sidebar">
		<div class="brand">
			<div class="brand-mark">
				<span class="icon-[solar--shield-network-bold-duotone]"></span>
			</div>
			<div>
				<p class="eyebrow">For Little</p>
				<h1>Admin Dashboard</h1>
				<p>Machine-centric browsing governance for shared laptops.</p>
			</div>
		</div>

		<button
			type="button"
			class="close-sidebar"
			aria-label="Close navigation"
			onclick={closeSidebar}
		>
			Close
		</button>

		<nav class="nav">
			{#each navItems as item (item.href)}
				<a
					class:active={page.url.pathname === item.href}
					href={resolve(item.href)}
					onclick={closeSidebar}
				>
					<span class={item.icon}></span>
					<div>
						<strong>{item.label}</strong>
						<small>{item.description}</small>
					</div>
				</a>
			{/each}
		</nav>
	</aside>

	<main class="content">
		<header class="page-header">
			<div class="page-header__eyebrow">
				<span class="icon-[solar--pulse-2-bold-duotone]"></span>
				<p class="eyebrow">Phase 1</p>
			</div>
			<div class="page-header__copy">
				<h2>{title}</h2>
				<p>{summary}</p>
			</div>
		</header>

		<section class="panel">{@render children()}</section>
	</main>
</div>
