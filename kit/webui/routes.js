export const Routes = {
    off: 'off',
    bidule: 'bidule',
    samples: 'samplesplitter',
    both: 'both'
};

export function routeLabel(route) {
    return route === Routes.samples ? 'Prophecy' :
        route === Routes.bidule ? 'Bidule' :
        route === Routes.both ? 'Both' :
        'Off';
}

export function initialPageDefaultRoute(initialPage) {
    return initialPage === 'pro' ? Routes.bidule : Routes.samples;
}
