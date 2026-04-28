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
		<div class="rounded-2xl border border-black/10 bg-white/95 p-5">
			<p class="m-0 leading-relaxed text-[#555555]">Loading machines from the backend...</p>
		</div>
	{:else if errorMessage}
		<div class="rounded-2xl border border-black/10 bg-white/95 p-5">
			<p class="m-0 leading-relaxed text-[#555555]">{errorMessage}</p>
		</div>
	{:else if machines.length === 0}
		<div class="rounded-2xl border border-black/10 bg-white/95 p-5">
			<p class="m-0 leading-relaxed text-[#555555]">No machines have registered yet.</p>
		</div>
	{:else}
		<div class="grid gap-4">
			{#each machines as machine (machine.id)}
				<article
					class="rounded-2xl border border-black/10 bg-white/95 p-5 shadow-[0_16px_36px_rgba(0,0,0,0.05)]"
				>
					<div class="mb-4 flex items-start justify-between gap-4">
						<div>
							<p class="mb-2 text-xs font-bold tracking-[0.14em] text-[#3d3d3d] uppercase">
								Machine
							</p>
							<h3 class="m-0 text-xl font-semibold text-[#111111]">
								{machine.display_name || machine.machine_id}
							</h3>
							<p class="mt-2 text-sm text-[#555555]">Shared laptop activity source</p>
						</div>
						<span
							class={[
								'inline-flex items-center justify-center rounded-full px-3 py-1.5 text-xs font-bold capitalize',
								machine.status === 'active'
									? 'bg-[#496b3a]/15 text-[#25441f]'
									: 'bg-[#496b3a]/10 text-[#25441f]'
							]}
						>
							{machine.status}
						</span>
					</div>

					<dl class="grid gap-3 md:grid-cols-3">
						<div>
							<span
								class="mb-2 icon-[solar--hashtag-square-bold-duotone] inline-flex text-lg text-[#25441f]"
							></span>
							<dt class="text-xs font-bold tracking-[0.1em] text-[#555555] uppercase">
								Machine ID
							</dt>
							<dd class="mt-1 break-words text-[#111111]">{machine.machine_id}</dd>
						</div>
						<div>
							<span
								class="mb-2 icon-[solar--clock-circle-bold-duotone] inline-flex text-lg text-[#25441f]"
							></span>
							<dt class="text-xs font-bold tracking-[0.1em] text-[#555555] uppercase">Last Seen</dt>
							<dd class="mt-1 text-[#111111]">{formatDate(machine.last_seen_at)}</dd>
						</div>
						<div>
							<span
								class="mb-2 icon-[solar--calendar-mark-bold-duotone] inline-flex text-lg text-[#25441f]"
							></span>
							<dt class="text-xs font-bold tracking-[0.1em] text-[#555555] uppercase">Created</dt>
							<dd class="mt-1 text-[#111111]">{formatDate(machine.created_at)}</dd>
						</div>
					</dl>
				</article>
			{/each}
		</div>
	{/if}
</AdminShell>
