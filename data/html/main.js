angular.module('main', ['ngMaterial', 'ui.router']);

// change angularjs interpoles to {[stuff]} rather than {{stuff}}
// angular.module('main').config(function($interpolateProvider) {
//     $interpolateProvider.startSymbol('{[').endSymbol(']}');
// });

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

    $rootScope.toast = function(text) {
        console.log("notify: ", text);
        $mdToast.showSimple(text);
    };

    $rootScope.updateUrl = function(args) {
        if ($rootScope.state == null) {
            return
        }

        $state.transitionTo($rootScope.state.name, args, {notify: false});
    }

    $rootScope.$on('$stateChangeStart', function(event, toState, toParams, fromState, fromParams, options) {});
    $rootScope.$on('$stateChangeError', function(event, toState, toParams, fromState, fromParams, error) {
        console.log(error);
    });

    $rootScope.$on('$stateChangeSuccess', function(event, toState, toParams, fromState, fromParams) {
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

angular.module('main').config(function($stateProvider, $urlRouterProvider, $locationProvider) {
    $urlRouterProvider.otherwise("/");
    $stateProvider
        .state('root', { abstract: true, template: '<ui-view/>' })
        // .state('root.search', { data: { title: 'Search' }, url: '/search?q&tags&authors', templateUrl: '/tmpl/search.html', controller: 'searchCtrl' })
        .state('root.home', { data: { title: 'Test Results' }, url: '/?q', templateUrl: '/index.html', controller: 'mainCtrl' })
        .state('root.test', { data: { title: 'TESTING' }, url: '/test', templateUrl: '/test.html' })
        .state('root.raw', { data: { title: 'Raw Crawl Results' }, url: '/raw/data', templateUrl: '/raw.html' })
});

angular.module('main').controller('mainCtrl', function ($scope, $rootScope, $stateParams) {
    $scope.urlViewing = -1;
    $scope.q = $stateParams.q;
    $scope.setURL = function (index) {
        if ($scope.urlViewing == index) {
            $scope.urlViewing = -1;
            return
        }

        $scope.urlViewing = index;
    }

    $scope.qfilter = function (item) {
        if ($scope.q == "" || $scope.q == null) { return true; }

        if (item.Result.URL.includes($scope.q)) { return true; }

        if (item.Result.Request != null) {
            if (item.Result.Request.IP.includes($scope.q)) { return true; }
        }

        if (angular.isNumber($scope.q) && parseFloat($scope.q) >= item.Score) { return true; }
        if (item.ErrorString.includes($scope.q)) { return true; }

        return false;
    }

    $scope.$watch("q", function() {
        $rootScope.updateUrl({ q: $scope.q });
    });

    console.log($rootScope.data);
});

angular.module('main').filter('prettyJSON', function() {
    return function(json) { return angular.toJson(json, true); }
});

function capitalizeFirstLetter(string) {
    return string.charAt(0).toUpperCase() + string.slice(1);
}
