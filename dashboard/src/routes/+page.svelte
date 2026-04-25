<script lang="ts">
	import AdminShell from '$lib/components/AdminShell.svelte';
	import { appConfig } from '$lib/config';

	const nextSteps = [
		'Show registered machines with display names and last seen timestamps',
		'Load real browsing logs grouped by machine activity',
		'Render global block and allow rules from the Go server',
		'Keep managed-machine rollout ready after local validation'
	];

	const principles = [
		'The machine is the primary tracked entity.',
		'Logs answer which machine opened which page and when.',
		'Rules stay simple first: allowlist and blocklist before anything smarter.'
	];
</script>

<svelte:head>
	<title>For Little Dashboard</title>
</svelte:head>

<AdminShell
	title="Overview"
	summary="This dashboard is the operating console for shared machines, browsing rules, and visit logs."
>
	<div class="overview-grid">
		<article class="card">
			<div class="card-icon icon-[solar--server-square-cloud-bold-duotone]"></div>
			<h3>Current Stack</h3>
			<dl>
				<div>
					<dt>Frontend</dt>
					<dd>SvelteKit</dd>
				</div>
				<div>
					<dt>Backend</dt>
					<dd>Go + Gin + PostgreSQL</dd>
				</div>
				<div>
					<dt>API Base URL</dt>
					<dd>{appConfig.apiBaseUrl}</dd>
				</div>
			</dl>
		</article>

		<article class="card">
			<div class="card-icon icon-[solar--checklist-minimalistic-bold-duotone]"></div>
			<h3>Next Integration Targets</h3>
			<ul>
				{#each nextSteps as item (item)}
					<li>{item}</li>
				{/each}
			</ul>
		</article>

		<article class="card card--wide">
			<div class="card-icon icon-[solar--notes-minimalistic-bold-duotone]"></div>
			<h3>Operating Principles</h3>
			<ul>
				{#each principles as item (item)}
					<li>{item}</li>
				{/each}
			</ul>
		</article>
	</div>
</AdminShell>
