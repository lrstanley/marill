angular.module('main', ['ngMaterial', 'ui.router']);

// change angularjs interpoles to {[stuff]} rather than {{stuff}}
// angular.module('main').config(function($interpolateProvider) {
//     $interpolateProvider.startSymbol('{[').endSymbol(']}');
// });

angular.module('main').run(function ($rootScope, $mdToast, $state) {
    $rootScope.pagetitle = "--";

    $rootScope.$watch("state.url", function () {
        if ($rootScope.state == undefined) { return }

        var parts = $rootScope.state.url.split("/");

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

        console.log($rootScope.pageurl);
    });  

    $rootScope.title = function (text) {
        $rootScope.pagetitle = text;
        window.document.title = text + ' - Marill - Automated Site Testing Utility';
    }

    $rootScope.toast = function(text) {
        console.log("toast: ", text);
        $mdToast.showSimple(text);
    };

    $rootScope.updateUrl = function(args) {
        return $state.transitionTo($rootScope.state.name, args, {notify: false});
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
        console.log("ERRORS!");
    }
});

angular.module('main').config(function($stateProvider, $urlRouterProvider, $locationProvider) {
    $urlRouterProvider.otherwise("/");
    $stateProvider
        .state('root', { abstract: true, template: '<ui-view/>' })
        // .state('root.search', { data: { title: 'Search' }, url: '/search?q&tags&authors', templateUrl: '/tmpl/search.html', controller: 'searchCtrl' })
        .state('root.home', { data: { title: 'Test Results' }, url: '/', templateUrl: '/index.html', controller: 'mainCtrl' })
        .state('root.test', { data: { title: 'TESTING' }, url: '/test', templateUrl: '/test.html' })
        .state('root.raw', { data: { title: 'Raw Crawl Results' }, url: '/raw/data', templateUrl: '/raw.html' })
});

angular.module('main').controller('mainCtrl', function($scope, $rootScope) {
    console.log($rootScope.data);
});

angular.module('main').filter('prettyJSON', function() {
    return function(json) { return angular.toJson(json, true); }
});

function capitalizeFirstLetter(string) {
    return string.charAt(0).toUpperCase() + string.slice(1);
}