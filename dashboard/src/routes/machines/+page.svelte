<script lang="ts">
	import AdminShell from '$lib/components/AdminShell.svelte';
	import { apiFetch } from '$lib/api';
	import type { Machine } from '$lib/types';
	import { onMount } from 'svelte';

	let machines = $state<Machine[]>([]);
	let loading = $state(true);
	let errorMessage = $state('');

	function formatDate(value: string | null) {
		if (!value) return 'Never seen';
		return new Date(value).toLocaleString();
	}

	onMount(async () => {
		try {
			machines = await apiFetch<Machine[]>('/api/v1/admin/machines');
		} catch (error) {
			errorMessage = error instanceof Error ? error.message : 'Could not load machines.';
		} finally {
			loading = false;
		}
	});
</script>

<svelte:head>
	<title>Machines | For Little</title>
</svelte:head>

<AdminShell
	title="Machines"
	summary="This page tracks which laptops are enrolled, how they are named, and when they last checked in."
>
	{#if loading}
		<div class="placeholder">
			<p>Loading machines from the backend...</p>
		</div>
	{:else if errorMessage}
		<div class="placeholder">
			<p>{errorMessage}</p>
		</div>
	{:else if machines.length === 0}
		<div class="placeholder">
			<p>No machines have registered yet.</p>
		</div>
	{:else}
		<div class="machine-list">
			{#each machines as machine (machine.id)}
				<article class="machine-card">
					<div class="machine-card__header">
						<div>
							<p class="eyebrow">Machine</p>
							<h3>{machine.display_name || machine.machine_id}</h3>
							<p class="machine-card__subtitle">Shared laptop activity source</p>
						</div>
						<span class:pending={machine.status !== 'active'} class="status-pill"
							>{machine.status}</span
						>
					</div>

					<dl class="machine-meta">
						<div>
							<span class="machine-meta__icon icon-[solar--hashtag-square-bold-duotone]"></span>
							<dt>Machine ID</dt>
							<dd>{machine.machine_id}</dd>
						</div>
						<div>
							<span class="machine-meta__icon icon-[solar--clock-circle-bold-duotone]"></span>
							<dt>Last Seen</dt>
							<dd>{formatDate(machine.last_seen_at)}</dd>
						</div>
						<div>
							<span class="machine-meta__icon icon-[solar--calendar-mark-bold-duotone]"></span>
							<dt>Created</dt>
							<dd>{formatDate(machine.created_at)}</dd>
						</div>
					</dl>
				</article>
			{/each}
		</div>
	{/if}
</AdminShell>
