angular.module('main', ['ngMaterial', 'ui.router', 'chart.js']);

angular.module('main').run(function ($rootScope, $mdToast, $state) {
    $rootScope.pagetitle = "--";

    $rootScope.$watch("state.url", function () {
        if ($rootScope.state == undefined) { return }

        var uri = $rootScope.state.url;
        if (uri.indexOf("?") != -1) {
            uri = uri.slice(0, uri.indexOf("?"))
        }

        var parts = uri.split("/");
        var out = [];

        for (i = 0; i < parts.length; i++) {
            if (!parts[i]) { continue }

            out.push(capitalizeFirstLetter(parts[i]))
        }

        if (out.length == 0) {
            $rootScope.pageurl = "/ Dashboard";
            return
        }

        $rootScope.pageurl = "/ " + out.join(" / ");
    });

    $rootScope.title = function (text) {
        $rootScope.pagetitle = text;
        window.document.title = text + ' - Marill - Automated Site Testing Utility';
    }

    $rootScope.toast = function (text) {
        console.log("notify: ", text);
        $mdToast.showSimple(text);
    };

    $rootScope.updateUrl = function (args) {
        if ($rootScope.state == null) {
            return
        }

        $state.transitionTo($rootScope.state.name, args, {notify: false});
    }

    $rootScope.$on('$stateChangeStart', function (event, toState, toParams, fromState, fromParams, options) {});
    $rootScope.$on('$stateChangeError', function (event, toState, toParams, fromState, fromParams, error) {
        console.log(error);
    });

    $rootScope.$on('$stateChangeSuccess', function (event, toState, toParams, fromState, fromParams) {
        console.log(`state-redirect: ${fromState.name} => ${toState.name}`);

        $rootScope.state = toState;
        $rootScope.title(toState.data.title);
    });

    $rootScope.data = JSON.parse(document.getElementById('data').innerHTML);
    if (!$rootScope.data.Success) {
        // some kind of error occurred.
        console.log("Error parsing embedded json");
    }
});

angular.module('main').config(function ($stateProvider, $urlRouterProvider, $locationProvider) {
    $urlRouterProvider.otherwise("/");
    $stateProvider
        .state('root', { abstract: true, template: '<ui-view/>' })
        .state('root.home', { data: { title: 'Test Results', rtype: 'all' }, url: '/?q', templateUrl: '/index.html', controller: 'mainCtrl' })
        .state('root.success', { data: { title: 'Successful Results', rtype: 'success' }, url: '/results/success?q', templateUrl: '/index.html', controller: 'mainCtrl' })
        .state('root.failed', { data: { title: 'Failed Results', rtype: 'failed' }, url: '/results/failed?q', templateUrl: '/index.html', controller: 'mainCtrl' })
        .state('root.test', { data: { title: 'TESTING' }, url: '/test', templateUrl: '/test.html' })
        .state('root.raw', { data: { title: 'Raw Crawl Results' }, url: '/raw/data', templateUrl: '/raw.html' })
});

angular.module('main').controller('mainCtrl', function ($scope, $rootScope, $state, $stateParams, $filter) {
    $scope.urlViewing = -1;
    $scope.q = $stateParams.q;

    $scope.setURL = function (index) {
        if ($scope.urlViewing == index) {
            $scope.urlViewing = -1;
            $scope.curResult = {};
            return
        }

        $scope.curResult = $scope.getAssetStats(index);
        $scope.urlViewing = index;
    }

    $scope.getAssetStats = function (index) {
        var out = { labels: [], data: [] };
        if (index == -1 || $rootScope.data.Out[index].Assets == null) {
            return out;
        }

        var tmp = {};

        for (i = 0; i < $rootScope.data.Out[index].Assets.length; i++) {
            var ctype = $rootScope.data.Out[index].Assets[i].ContentType;
            if (ctype.indexOf(";") != -1) {
                ctype = ctype.split(0, ctype.indexOf(";"));
            }

            if (tmp[ctype] == null) { tmp[ctype] = 0; }
            tmp[ctype] += 1;
        }

        for (var ctype in tmp) {
            out.labels.push(ctype);
            out.data.push(tmp[ctype]);
        }

        return out;
    }
    $scope.curResult = $scope.getAssetStats(-1);

    $scope.assetStatus = function (asset) {
        var out = "Status: " + asset.Code;

        if (asset.ContentLength > 0) {
            out += " Size: " + $filter('number')(asset.ContentLength / 1024, 2) + "kb";
        }

        if (asset.Error.length) {
            out += " Error: " + asset.Error;
        }

        return out
    }

    $scope.qfilter = function (item) {
        if ($state.current.data.rtype != null && $state.current.data.rtype != 'none') {
            if ($state.current.data.rtype == 'success' && (item.ErrorString != "" || item.Score < $rootScope.data.MinScore)) { return false; }
            if ($state.current.data.rtype == 'failed' && (item.ErrorString == "" && item.Score >= $rootScope.data.MinScore && item.Result.Response != null)) { return false; }
        }

        if ($scope.q == "" || $scope.q == null) { return true; }

        if (item.Result.URL.includes($scope.q)) { return true; }

        if (item.Result.Request != null) {
            if (item.Result.Request.IP.includes($scope.q)) { return true; }
        }

        if (angular.isNumber($scope.q) && parseFloat($scope.q) >= item.Score) { return true; }
        if (item.ErrorString.includes($scope.q)) { return true; }

        return false;
    }

    $scope.$watch("q", function () { $rootScope.updateUrl({ q: $scope.q }); });

    console.log($rootScope.data);
});

angular.module('main').filter('prettyJSON', function () {
    return function (json) { return angular.toJson(json, true); }
});

function capitalizeFirstLetter(string) {
    return string.charAt(0).toUpperCase() + string.slice(1);
}
