import {id, version} from './manifest';

// To ease migration, verify separate export of id and version.
test('Plugin id and version are defined', () => {
    expect(id).toBeDefined();
    expect(version).toBeDefined();
});
