// Code generated by counterfeiter. DO NOT EDIT.
package fakes

import (
	"sync"

	"github.com/weaveworks/pctl/pkg/catalog"
	"github.com/weaveworks/profiles/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type FakeCatalogManager struct {
	InstallStub        func(catalog.InstallConfig) error
	installMutex       sync.RWMutex
	installArgsForCall []struct {
		arg1 catalog.InstallConfig
	}
	installReturns struct {
		result1 error
	}
	installReturnsOnCall map[int]struct {
		result1 error
	}
	ListStub        func(client.Client, catalog.CatalogClient) ([]catalog.ProfileData, error)
	listMutex       sync.RWMutex
	listArgsForCall []struct {
		arg1 client.Client
		arg2 catalog.CatalogClient
	}
	listReturns struct {
		result1 []catalog.ProfileData
		result2 error
	}
	listReturnsOnCall map[int]struct {
		result1 []catalog.ProfileData
		result2 error
	}
	SearchStub        func(catalog.CatalogClient, string) ([]v1alpha1.ProfileCatalogEntry, error)
	searchMutex       sync.RWMutex
	searchArgsForCall []struct {
		arg1 catalog.CatalogClient
		arg2 string
	}
	searchReturns struct {
		result1 []v1alpha1.ProfileCatalogEntry
		result2 error
	}
	searchReturnsOnCall map[int]struct {
		result1 []v1alpha1.ProfileCatalogEntry
		result2 error
	}
	ShowStub        func(catalog.CatalogClient, string, string, string) (v1alpha1.ProfileCatalogEntry, error)
	showMutex       sync.RWMutex
	showArgsForCall []struct {
		arg1 catalog.CatalogClient
		arg2 string
		arg3 string
		arg4 string
	}
	showReturns struct {
		result1 v1alpha1.ProfileCatalogEntry
		result2 error
	}
	showReturnsOnCall map[int]struct {
		result1 v1alpha1.ProfileCatalogEntry
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeCatalogManager) Install(arg1 catalog.InstallConfig) error {
	fake.installMutex.Lock()
	ret, specificReturn := fake.installReturnsOnCall[len(fake.installArgsForCall)]
	fake.installArgsForCall = append(fake.installArgsForCall, struct {
		arg1 catalog.InstallConfig
	}{arg1})
	stub := fake.InstallStub
	fakeReturns := fake.installReturns
	fake.recordInvocation("Install", []interface{}{arg1})
	fake.installMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeCatalogManager) InstallCallCount() int {
	fake.installMutex.RLock()
	defer fake.installMutex.RUnlock()
	return len(fake.installArgsForCall)
}

func (fake *FakeCatalogManager) InstallCalls(stub func(catalog.InstallConfig) error) {
	fake.installMutex.Lock()
	defer fake.installMutex.Unlock()
	fake.InstallStub = stub
}

func (fake *FakeCatalogManager) InstallArgsForCall(i int) catalog.InstallConfig {
	fake.installMutex.RLock()
	defer fake.installMutex.RUnlock()
	argsForCall := fake.installArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeCatalogManager) InstallReturns(result1 error) {
	fake.installMutex.Lock()
	defer fake.installMutex.Unlock()
	fake.InstallStub = nil
	fake.installReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeCatalogManager) InstallReturnsOnCall(i int, result1 error) {
	fake.installMutex.Lock()
	defer fake.installMutex.Unlock()
	fake.InstallStub = nil
	if fake.installReturnsOnCall == nil {
		fake.installReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.installReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeCatalogManager) List(arg1 client.Client, arg2 catalog.CatalogClient) ([]catalog.ProfileData, error) {
	fake.listMutex.Lock()
	ret, specificReturn := fake.listReturnsOnCall[len(fake.listArgsForCall)]
	fake.listArgsForCall = append(fake.listArgsForCall, struct {
		arg1 client.Client
		arg2 catalog.CatalogClient
	}{arg1, arg2})
	stub := fake.ListStub
	fakeReturns := fake.listReturns
	fake.recordInvocation("List", []interface{}{arg1, arg2})
	fake.listMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeCatalogManager) ListCallCount() int {
	fake.listMutex.RLock()
	defer fake.listMutex.RUnlock()
	return len(fake.listArgsForCall)
}

func (fake *FakeCatalogManager) ListCalls(stub func(client.Client, catalog.CatalogClient) ([]catalog.ProfileData, error)) {
	fake.listMutex.Lock()
	defer fake.listMutex.Unlock()
	fake.ListStub = stub
}

func (fake *FakeCatalogManager) ListArgsForCall(i int) (client.Client, catalog.CatalogClient) {
	fake.listMutex.RLock()
	defer fake.listMutex.RUnlock()
	argsForCall := fake.listArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeCatalogManager) ListReturns(result1 []catalog.ProfileData, result2 error) {
	fake.listMutex.Lock()
	defer fake.listMutex.Unlock()
	fake.ListStub = nil
	fake.listReturns = struct {
		result1 []catalog.ProfileData
		result2 error
	}{result1, result2}
}

func (fake *FakeCatalogManager) ListReturnsOnCall(i int, result1 []catalog.ProfileData, result2 error) {
	fake.listMutex.Lock()
	defer fake.listMutex.Unlock()
	fake.ListStub = nil
	if fake.listReturnsOnCall == nil {
		fake.listReturnsOnCall = make(map[int]struct {
			result1 []catalog.ProfileData
			result2 error
		})
	}
	fake.listReturnsOnCall[i] = struct {
		result1 []catalog.ProfileData
		result2 error
	}{result1, result2}
}

func (fake *FakeCatalogManager) Search(arg1 catalog.CatalogClient, arg2 string) ([]v1alpha1.ProfileCatalogEntry, error) {
	fake.searchMutex.Lock()
	ret, specificReturn := fake.searchReturnsOnCall[len(fake.searchArgsForCall)]
	fake.searchArgsForCall = append(fake.searchArgsForCall, struct {
		arg1 catalog.CatalogClient
		arg2 string
	}{arg1, arg2})
	stub := fake.SearchStub
	fakeReturns := fake.searchReturns
	fake.recordInvocation("Search", []interface{}{arg1, arg2})
	fake.searchMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeCatalogManager) SearchCallCount() int {
	fake.searchMutex.RLock()
	defer fake.searchMutex.RUnlock()
	return len(fake.searchArgsForCall)
}

func (fake *FakeCatalogManager) SearchCalls(stub func(catalog.CatalogClient, string) ([]v1alpha1.ProfileCatalogEntry, error)) {
	fake.searchMutex.Lock()
	defer fake.searchMutex.Unlock()
	fake.SearchStub = stub
}

func (fake *FakeCatalogManager) SearchArgsForCall(i int) (catalog.CatalogClient, string) {
	fake.searchMutex.RLock()
	defer fake.searchMutex.RUnlock()
	argsForCall := fake.searchArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeCatalogManager) SearchReturns(result1 []v1alpha1.ProfileCatalogEntry, result2 error) {
	fake.searchMutex.Lock()
	defer fake.searchMutex.Unlock()
	fake.SearchStub = nil
	fake.searchReturns = struct {
		result1 []v1alpha1.ProfileCatalogEntry
		result2 error
	}{result1, result2}
}

func (fake *FakeCatalogManager) SearchReturnsOnCall(i int, result1 []v1alpha1.ProfileCatalogEntry, result2 error) {
	fake.searchMutex.Lock()
	defer fake.searchMutex.Unlock()
	fake.SearchStub = nil
	if fake.searchReturnsOnCall == nil {
		fake.searchReturnsOnCall = make(map[int]struct {
			result1 []v1alpha1.ProfileCatalogEntry
			result2 error
		})
	}
	fake.searchReturnsOnCall[i] = struct {
		result1 []v1alpha1.ProfileCatalogEntry
		result2 error
	}{result1, result2}
}

func (fake *FakeCatalogManager) Show(arg1 catalog.CatalogClient, arg2 string, arg3 string, arg4 string) (v1alpha1.ProfileCatalogEntry, error) {
	fake.showMutex.Lock()
	ret, specificReturn := fake.showReturnsOnCall[len(fake.showArgsForCall)]
	fake.showArgsForCall = append(fake.showArgsForCall, struct {
		arg1 catalog.CatalogClient
		arg2 string
		arg3 string
		arg4 string
	}{arg1, arg2, arg3, arg4})
	stub := fake.ShowStub
	fakeReturns := fake.showReturns
	fake.recordInvocation("Show", []interface{}{arg1, arg2, arg3, arg4})
	fake.showMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2, arg3, arg4)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeCatalogManager) ShowCallCount() int {
	fake.showMutex.RLock()
	defer fake.showMutex.RUnlock()
	return len(fake.showArgsForCall)
}

func (fake *FakeCatalogManager) ShowCalls(stub func(catalog.CatalogClient, string, string, string) (v1alpha1.ProfileCatalogEntry, error)) {
	fake.showMutex.Lock()
	defer fake.showMutex.Unlock()
	fake.ShowStub = stub
}

func (fake *FakeCatalogManager) ShowArgsForCall(i int) (catalog.CatalogClient, string, string, string) {
	fake.showMutex.RLock()
	defer fake.showMutex.RUnlock()
	argsForCall := fake.showArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3, argsForCall.arg4
}

func (fake *FakeCatalogManager) ShowReturns(result1 v1alpha1.ProfileCatalogEntry, result2 error) {
	fake.showMutex.Lock()
	defer fake.showMutex.Unlock()
	fake.ShowStub = nil
	fake.showReturns = struct {
		result1 v1alpha1.ProfileCatalogEntry
		result2 error
	}{result1, result2}
}

func (fake *FakeCatalogManager) ShowReturnsOnCall(i int, result1 v1alpha1.ProfileCatalogEntry, result2 error) {
	fake.showMutex.Lock()
	defer fake.showMutex.Unlock()
	fake.ShowStub = nil
	if fake.showReturnsOnCall == nil {
		fake.showReturnsOnCall = make(map[int]struct {
			result1 v1alpha1.ProfileCatalogEntry
			result2 error
		})
	}
	fake.showReturnsOnCall[i] = struct {
		result1 v1alpha1.ProfileCatalogEntry
		result2 error
	}{result1, result2}
}

func (fake *FakeCatalogManager) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.installMutex.RLock()
	defer fake.installMutex.RUnlock()
	fake.listMutex.RLock()
	defer fake.listMutex.RUnlock()
	fake.searchMutex.RLock()
	defer fake.searchMutex.RUnlock()
	fake.showMutex.RLock()
	defer fake.showMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeCatalogManager) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ catalog.CatalogManager = new(FakeCatalogManager)
