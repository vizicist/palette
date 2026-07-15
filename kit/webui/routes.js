export const Routes = {
    off: 'off',
    bidule: 'bidule',
    samples: 'samplesplitter',
    both: 'both'
};

export function routeLabel(route) {
    return route === Routes.samples ? 'Words' :
        route === Routes.bidule ? 'Bidule' :
        route === Routes.both ? 'Both' :
        'Off';
}

export function initialPageDefaultRoute(initialPage) {
    // bss defaults to sample playback; pro and pro2 default to Bidule.
    return initialPage === 'bss' ? Routes.samples : Routes.bidule;
}
